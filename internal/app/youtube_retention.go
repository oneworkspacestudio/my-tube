package app

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Existing sources stay unlimited. Only uploads explicitly created by the
// importer acquire deletion ownership; deduplicated library videos never do.
func migrateYouTubeRetention(db *sql.DB) error {
	var version int
	if e := db.QueryRow("SELECT MAX(version) FROM migrations").Scan(&version); e != nil {
		return e
	}
	if version >= 4 {
		return nil
	}
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`ALTER TABLE youtube_sources ADD COLUMN retention_limit INTEGER NOT NULL DEFAULT 0;
ALTER TABLE youtube_items ADD COLUMN seen_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE youtube_items ADD COLUMN source_position INTEGER NOT NULL DEFAULT 0;
CREATE TABLE youtube_media(video_id TEXT PRIMARY KEY REFERENCES videos(id) ON DELETE CASCADE,
 source_id TEXT NOT NULL REFERENCES youtube_sources(id) ON DELETE CASCADE, owner TEXT NOT NULL REFERENCES users(id));
CREATE TABLE youtube_purge(video_id TEXT PRIMARY KEY REFERENCES videos(id) ON DELETE CASCADE,
 owner TEXT NOT NULL, original TEXT NOT NULL, source_id TEXT NOT NULL);
INSERT INTO migrations VALUES(4,0);`)
	if e != nil {
		return e
	}
	return tx.Commit()
}

func validYouTubeRetention(n int) bool { return n == 0 || n == 10 || n == 20 || n == 30 || n == 50 }
func validYouTubeSchedule(n int) bool  { return n == 0 || n == 1 || n == 6 || n == 24 || n == 168 }
func nextYouTubeCheck(at int64, hours int) int64 {
	if hours == 0 {
		return 0
	}
	return at + int64(hours)*3600000
}

// Called inside the link/settings transaction while writeMu is held. Ranking
// comes from the source's observed order, not download/processing completion.
func trimYouTube(tx *sql.Tx, source youtubeSource) error {
	if source.Retention == 0 {
		return nil
	}
	rows, e := tx.Query(`SELECT i.video_id FROM youtube_items i JOIN videos v ON v.id=i.video_id
 WHERE i.source_id=? AND i.state='saved' AND v.owner=? AND v.deleted=0
 GROUP BY i.video_id ORDER BY MAX(i.seen_at) DESC,
 CASE WHEN ? THEN -MAX(i.source_position) ELSE MIN(i.source_position) END, v.created DESC, v.id`, source.ID, source.Owner, source.Kind == "playlist" && source.PlaylistOrder == "last")
	if e != nil {
		return e
	}
	var ids []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			break
		}
		ids = append(ids, id)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return e
	}
	for index, id := range ids {
		if index < source.Retention {
			continue
		}
		if _, e = tx.Exec("DELETE FROM playlist_videos WHERE playlist_id=? AND video_id=?", source.Playlist, id); e != nil {
			return e
		}
		// Other source references, deliberate saves and shared/public media stay.
		var owned, protected int
		if e = tx.QueryRow("SELECT count(*) FROM youtube_media WHERE video_id=? AND owner=?", id, source.Owner).Scan(&owned); e != nil {
			return e
		}
		if e = tx.QueryRow(`SELECT
 (SELECT count(*) FROM youtube_items WHERE video_id=? AND source_id<>? AND state='saved')+
 (SELECT count(*) FROM playlist_videos WHERE video_id=?)+
 (SELECT count(*) FROM later WHERE video_id=?)+
 (SELECT count(*) FROM video_access WHERE video_id=?)+
 (SELECT count(*) FROM videos WHERE id=? AND visibility<>'private')`, id, source.ID, id, id, id, id).Scan(&protected); e != nil {
			return e
		}
		message := "Removed from this import by its storage limit; library copy kept"
		if owned == 1 && protected == 0 {
			if _, e = tx.Exec(`INSERT OR IGNORE INTO youtube_purge(video_id,owner,original,source_id)
 SELECT id,owner,original,? FROM videos WHERE id=? AND owner=?`, source.ID, id, source.Owner); e != nil {
				return e
			}
			if _, e = tx.Exec("UPDATE videos SET deleted=1,visibility='private' WHERE id=? AND owner=?", id, source.Owner); e != nil {
				return e
			}
			message = "Expired by storage limit; local download removal queued"
		}
		if _, e = tx.Exec("UPDATE youtube_items SET video_id='',upload_id='',state='expired',message=?,updated=? WHERE source_id=? AND video_id=?", message, now(), source.ID, id); e != nil {
			return e
		}
	}
	return nil
}

// Validate the complete deletion set before removing any byte. No links or
// unexpected files are traversed, even if private storage changes after startup.
func youtubeRemovalPaths(root, id, original string) ([]string, error) {
	if !objectID.MatchString(id) || original != id+".mp4" {
		return nil, errors.New("invalid imported media path")
	}
	absolute, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	root = absolute
	paths := []string{filepath.Join(root, "originals", original), filepath.Join(root, "streaming", id), filepath.Join(root, "thumbnails", id+".jpg"), filepath.Join(root, "subtitles", id+".vtt")}
	for index, target := range paths {
		for p := target; ; p = filepath.Dir(p) {
			info, e := os.Lstat(p)
			if e != nil && !os.IsNotExist(e) {
				return nil, e
			}
			if e == nil && (info.Mode()&os.ModeSymlink != 0 || youtubeReparse(p)) {
				return nil, errors.New("import cleanup refuses linked storage")
			}
			if filepath.Dir(p) == p {
				break
			}
		}
		if info, err := os.Lstat(target); err == nil && ((index == 1 && !info.IsDir()) || (index != 1 && !info.Mode().IsRegular())) {
			return nil, errors.New("unexpected imported media file type")
		}
		if e := filepath.WalkDir(target, func(p string, d os.DirEntry, e error) error {
			if os.IsNotExist(e) {
				return nil
			}
			if e != nil {
				return e
			}
			if d.Type()&os.ModeSymlink != 0 || youtubeReparse(p) || (!d.IsDir() && !d.Type().IsRegular()) {
				return errors.New("import cleanup refuses special files")
			}
			return nil
		}); e != nil {
			return nil, e
		}
	}
	return paths, nil
}

// Caller holds writeMu. Pending deletions survive restarts and wait for media
// processing or an open Windows playback handle rather than losing track of it.
func (a *App) purgeYouTubeLocked() error {
	rows, e := a.Store.DB.Query(`SELECT p.video_id,p.owner,p.original FROM youtube_purge p
 JOIN videos v ON v.id=p.video_id AND v.owner=p.owner AND v.original=p.original
 WHERE v.deleted=1 AND NOT EXISTS(SELECT 1 FROM jobs WHERE video_id=v.id AND status='running') LIMIT 4`)
	if e != nil {
		return e
	}
	type pending struct{ id, owner, original string }
	var list []pending
	for rows.Next() {
		var p pending
		if e = rows.Scan(&p.id, &p.owner, &p.original); e != nil {
			break
		}
		list = append(list, p)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return e
	}
	var failures []error
	for _, p := range list {
		paths, err := youtubeRemovalPaths(a.Config.MediaDir, p.id, p.original)
		if err == nil {
			for _, target := range paths {
				if err = os.RemoveAll(target); err != nil {
					break
				}
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("import cleanup pending: %w", err))
			continue
		}
		tx, err := a.Store.DB.Begin()
		if err != nil {
			return err
		}
		// These tables intentionally have restrictive foreign keys for normal UI
		// deletion. A proven importer-owned expiry is the explicit purge path.
		for _, table := range []string{"video_access", "likes", "later", "history", "comments", "uploads"} {
			if _, err = tx.Exec("DELETE FROM "+table+" WHERE video_id=?", p.id); err != nil {
				break
			}
		}
		if err == nil {
			_, err = tx.Exec("DELETE FROM videos WHERE id=? AND owner=? AND deleted=1", p.id, p.owner)
		}
		if err == nil {
			err = tx.Commit()
		} else {
			tx.Rollback()
		}
		if err != nil {
			tx.Rollback()
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// Legacy items have no trustworthy ownership marker. They may be deduplicated
// uploads, so expiry only detaches their source link and never erases originals.
func (a *App) youtubeCleanupSummary(source string) string {
	var count int
	if a.Store.DB.QueryRow("SELECT count(*) FROM youtube_purge WHERE source_id=?", source).Scan(&count) == nil && count > 0 {
		return " · Older downloads waiting for cleanup"
	}
	return ""
}
