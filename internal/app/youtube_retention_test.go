package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func retentionSource(t *testing.T, a *App, s session, handle string, limit int) youtubeSource {
	t.Helper()
	w := call(t, a, s, "POST", "/api/youtube", map[string]any{"url": "@" + handle, "retention_limit": limit, "auto_hours": 0}, nil)
	mustStatus(t, w, 202)
	id := result(t, w)["id"].(string)
	if _, e := a.Store.DB.Exec("UPDATE youtube_sources SET state='completed' WHERE id=?", id); e != nil {
		t.Fatal(e)
	}
	v, e := scanYouTube(a.Store.DB.QueryRow("SELECT "+youtubeSourceCols+" FROM youtube_sources WHERE id=?", id))
	if e != nil {
		t.Fatal(e)
	}
	return v
}

// Publish a unique valid MP4 through the real resumable upload/finalization path.
// Merely associating an ordinary upload later must not acquire purge ownership.
func retentionVideo(t *testing.T, a *App, s session, source youtubeSource, n int, managed bool) string {
	t.Helper()
	remote := fmt.Sprintf("v%010d", n)
	body, e := os.ReadFile("testdata/sample.mp4")
	if e != nil {
		t.Fatal(e)
	}
	body = append(body, 0, 0, 0, 12, 'f', 'r', 'e', 'e', byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	w := call(t, a, s, "POST", "/api/uploads", map[string]any{"name": remote + ".mp4", "title": remote, "size": len(body), "visibility": "private"}, nil)
	mustStatus(t, w, 201)
	id := result(t, w)["id"].(string)
	if managed {
		_, e = a.Store.DB.Exec("INSERT INTO youtube_items(source_id,remote_id,title,upload_id,seen_at,source_position,updated)VALUES(?,?,?,?,?,0,?)", source.ID, remote, remote, id, n, now())
		if e != nil {
			t.Fatal(e)
		}
	}
	mustStatus(t, call(t, a, s, "PATCH", "/api/uploads/"+id, body, map[string]string{"Upload-Offset": "0"}), 200)
	w = call(t, a, s, "POST", "/api/uploads/"+id+"/complete", map[string]any{}, nil)
	mustStatus(t, w, 201)
	video := result(t, w)["id"].(string)
	if !managed {
		_, e = a.Store.DB.Exec("INSERT INTO youtube_items(source_id,remote_id,title,seen_at,source_position,updated)VALUES(?,?,?,?,0,?)", source.ID, remote, remote, n, now())
		if e != nil {
			t.Fatal(e)
		}
	}
	if e = a.linkYouTube(source, remote, video, "Saved"); e != nil {
		t.Fatal(e)
	}
	return video
}

func sourceSaved(t *testing.T, a *App, source string) int {
	t.Helper()
	var n int
	if e := a.Store.DB.QueryRow("SELECT count(DISTINCT video_id) FROM youtube_items WHERE source_id=? AND state='saved' AND video_id<>''", source).Scan(&n); e != nil {
		t.Fatal(e)
	}
	return n
}

func TestYouTubeDefaultsSettingsAndOwnerBoundary(t *testing.T) {
	a := testApp(t)
	alice := login(t, a, "alice")
	bob := login(t, a, "bob")
	w := call(t, a, alice, "POST", "/api/youtube", map[string]any{"url": "@defaults"}, nil)
	mustStatus(t, w, 202)
	id := result(t, w)["id"].(string)
	v, e := scanYouTube(a.Store.DB.QueryRow("SELECT "+youtubeSourceCols+" FROM youtube_sources WHERE id=?", id))
	if e != nil {
		t.Fatal(e)
	}
	if v.Retention != 10 || v.Latest != 10 || v.AutoHours != 24 {
		t.Fatalf("wrong defaults: %+v", v)
	}
	mustStatus(t, call(t, a, alice, "PUT", "/api/youtube/"+id, map[string]any{"retention_limit": 20}, nil), 409)
	a.Store.DB.Exec("UPDATE youtube_sources SET state='completed' WHERE id=?", id)
	for _, limit := range []int{10, 20, 30, 50, 0} {
		for _, hours := range []int{0, 1, 6, 24, 168} {
			mustStatus(t, call(t, a, alice, "PUT", "/api/youtube/"+id, map[string]any{"retention_limit": limit, "auto_hours": hours}, nil), 200)
			v, e = scanYouTube(a.Store.DB.QueryRow("SELECT "+youtubeSourceCols+" FROM youtube_sources WHERE id=?", id))
			if e != nil {
				t.Fatal(e)
			}
			if v.Retention != limit || v.AutoHours != hours || (hours == 0 && v.Next != 0) || (hours > 0 && v.Next <= now()) {
				t.Fatalf("settings not saved: %+v", v)
			}
		}
	}
	for _, input := range []map[string]any{{"retention_limit": -1}, {"retention_limit": 11}, {"auto_hours": 3}} {
		mustStatus(t, call(t, a, alice, "PUT", "/api/youtube/"+id, input, nil), 400)
	}
	mustStatus(t, call(t, a, bob, "PUT", "/api/youtube/"+id, map[string]any{"retention_limit": 10}, nil), 404)
	// Cached older clients cannot accidentally enable a new deletion policy.
	mustStatus(t, call(t, a, alice, "PUT", "/api/youtube/"+id, map[string]any{"latest": 5, "auto_hours": 0}, nil), 200)
	v, e = scanYouTube(a.Store.DB.QueryRow("SELECT "+youtubeSourceCols+" FROM youtube_sources WHERE id=?", id))
	if e != nil || v.Retention != 0 {
		t.Fatal("legacy settings changed retention", e)
	}
}

func TestYouTubeRollingRetentionReclaimsDerivedFilesAfterReplacement(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	source := retentionSource(t, a, s, "rolling", 10)
	old := retentionVideo(t, a, s, source, 1, true)
	stream := filepath.Join(a.Config.MediaDir, "streaming", old)
	os.MkdirAll(stream, 0700)
	os.WriteFile(filepath.Join(stream, "q360.ts"), []byte("synthetic rendition"), 0600)
	os.WriteFile(filepath.Join(a.Config.MediaDir, "thumbnails", old+".jpg"), []byte("synthetic poster"), 0600)
	os.WriteFile(filepath.Join(a.Config.MediaDir, "subtitles", old+".vtt"), []byte("WEBVTT"), 0600)
	for n := 2; n <= 10; n++ {
		retentionVideo(t, a, s, source, n, true)
	}
	// A failed candidate must not evict a previously saved, usable video.
	a.Store.DB.Exec("INSERT INTO youtube_items(source_id,remote_id,title,state,seen_at,updated)VALUES(?,'failed00000','Unavailable','failed',100,?)", source.ID, now())
	mustStatus(t, call(t, a, s, "PUT", "/api/youtube/"+source.ID, map[string]any{"retention_limit": 10}, nil), 200)
	if _, e := os.Stat(filepath.Join(a.Config.MediaDir, "originals", old+".mp4")); e != nil {
		t.Fatal("failed replacement evicted original", e)
	}
	retentionVideo(t, a, s, source, 11, true)
	if got := sourceSaved(t, a, source.ID); got != 10 {
		t.Fatalf("saved %d, expected10", got)
	}
	for _, p := range []string{filepath.Join(a.Config.MediaDir, "originals", old+".mp4"), stream, filepath.Join(a.Config.MediaDir, "thumbnails", old+".jpg"), filepath.Join(a.Config.MediaDir, "subtitles", old+".vtt")} {
		if _, e := os.Stat(p); !os.IsNotExist(e) {
			t.Fatalf("expired bytes remain at %s: %v", p, e)
		}
	}
	var n int
	a.Store.DB.QueryRow("SELECT count(*) FROM videos WHERE id=?", old).Scan(&n)
	if n != 0 {
		t.Fatal("expired video row remains")
	}
}

func TestYouTubeRetentionPreservesOtherReferencesAndUploadedDuplicates(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	source := retentionSource(t, a, s, "protected", 10)
	other := retentionSource(t, a, s, "shared", 0)
	shared := retentionVideo(t, a, s, source, 1, true)
	a.Store.DB.Exec("INSERT INTO youtube_items(source_id,remote_id,title,updated)VALUES(?,'v0000000001','Shared',?)", other.ID, now())
	if e := a.linkYouTube(other, "v0000000001", shared, "Already saved"); e != nil {
		t.Fatal(e)
	}
	uploaded := retentionVideo(t, a, s, source, 2, false)
	saved := retentionVideo(t, a, s, source, 3, true)
	w := call(t, a, s, "POST", "/api/playlists", map[string]any{"title": "Keep this", "visibility": "private"}, nil)
	mustStatus(t, w, 200)
	playlist := result(t, w)["id"].(string)
	mustStatus(t, call(t, a, s, "POST", "/api/playlists/"+playlist+"/videos", map[string]any{"video_id": saved}, nil), 200)
	for n := 4; n <= 13; n++ {
		retentionVideo(t, a, s, source, n, true)
	}
	if sourceSaved(t, a, source.ID) != 10 || sourceSaved(t, a, other.ID) != 1 {
		t.Fatal("wrong retained source membership")
	}
	for _, id := range []string{shared, uploaded, saved} {
		if _, e := os.Stat(filepath.Join(a.Config.MediaDir, "originals", id+".mp4")); e != nil {
			t.Fatal("protected original removed", e)
		}
		mustStatus(t, call(t, a, s, "GET", "/api/videos/"+id, nil, nil), 200)
	}
}

func TestYouTubeUnlimitedAndPendingCleanupSurviveRestart(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	source := retentionSource(t, a, s, "unlimited", 0)
	old := retentionVideo(t, a, s, source, 1, true)
	for n := 2; n <= 12; n++ {
		retentionVideo(t, a, s, source, n, true)
	}
	if sourceSaved(t, a, source.ID) != 12 {
		t.Fatal("unlimited pruned saved videos")
	}
	a.Store.DB.Exec("UPDATE jobs SET status='running' WHERE video_id=?", old)
	mustStatus(t, call(t, a, s, "PUT", "/api/youtube/"+source.ID, map[string]any{"retention_limit": 10}, nil), 200)
	a.writeMu.Lock()
	e := a.purgeYouTubeLocked()
	a.writeMu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(a.Config.MediaDir, "originals", old+".mp4")); e != nil {
		t.Fatal("purged while processing", e)
	}
	if e = a.Store.Close(); e != nil {
		t.Fatal(e)
	}
	a.Store, e = OpenStore(a.Config.DataDir)
	if e != nil {
		t.Fatal(e)
	}
	a.Store.DB.Exec("UPDATE jobs SET status='ready' WHERE video_id=?", old)
	a.writeMu.Lock()
	e = a.purgeYouTubeLocked()
	a.writeMu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(a.Config.MediaDir, "originals", old+".mp4")); !os.IsNotExist(e) {
		t.Fatal("pending cleanup did not resume", e)
	}
}

func TestYouTubeSchedulerCatchesUpWithoutBrowserAndRespectsManual(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	scheduled := retentionSource(t, a, s, "scheduled", 10)
	manual := retentionSource(t, a, s, "manual", 10)
	a.Store.DB.Exec("UPDATE youtube_sources SET auto_hours=1,next_check=1 WHERE id=?", scheduled.ID)
	a.Store.DB.Exec("UPDATE youtube_sources SET next_check=1 WHERE id=?", manual.ID)
	for attempt := 0; attempt < 2; attempt++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		called := make(chan string, 1)
		go func() {
			defer close(done)
			a.youtubeWorkerLoop(ctx, func(ctx context.Context, source youtubeSource) error {
				a.Store.DB.Exec("UPDATE youtube_sources SET next_check=1 WHERE id=?", source.ID)
				a.youtubeStatus(source.ID, "completed", "Synthetic check", true)
				called <- source.ID
				return nil
			})
		}()
		select {
		case id := <-called:
			if id != scheduled.ID {
				t.Error("manual source ran automatically")
			}
		case <-time.After(5 * time.Second):
			t.Error("missed scheduled check")
		}
		cancel()
		<-done
		var next int64
		a.Store.DB.QueryRow("SELECT next_check FROM youtube_sources WHERE id=?", scheduled.ID).Scan(&next)
		if next <= now()+3500000 {
			t.Fatal("schedule did not advance after completion")
		}
		a.Store.DB.Exec("UPDATE youtube_sources SET next_check=1 WHERE id=?", scheduled.ID)
	}
}

func TestYouTubeSchema3MigrationPreservesExistingSources(t *testing.T) {
	dir := t.TempDir()
	db, e := sql.Open("sqlite", filepath.Join(dir, "mytube.db"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(schema); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(youtubeSchema); e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec(`INSERT INTO users(id,issuer,subject,name,email,avatar,quota_mb)VALUES('u','i','s','User','','',50);
INSERT INTO playlists(id,owner,title,created)VALUES('p','u','Legacy',1);
INSERT INTO youtube_sources(id,owner,url,kind,title,playlist_id,latest,auto_hours,state,created,next_check)
VALUES('s','u','https://www.youtube.com/@legacy/videos','channel','Legacy','p',5,24,'completed',1,123);
INSERT INTO channels(id,owner,name,handle,created)VALUES('c','u','Legacy','legacy',1);
INSERT INTO videos(id,owner,channel_id,title,visibility,original,filename,size,sha256,created)
VALUES('v','u','c','Existing saved video','private','v.mp4','original.mp4',100,'legacy-digest',1);
INSERT INTO youtube_items(source_id,remote_id,title,state,video_id,updated)
VALUES('s','v0000000001','Existing saved video','saved','v',1);
INSERT INTO playlist_videos(playlist_id,video_id,position)VALUES('p','v',1);`)
	if e != nil {
		t.Fatal(e)
	}
	db.Close()
	store, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	var limit, latest, hours int
	var next int64
	if e = store.DB.QueryRow("SELECT retention_limit,latest,auto_hours,next_check FROM youtube_sources WHERE id='s'").Scan(&limit, &latest, &hours, &next); e != nil {
		t.Fatal(e)
	}
	if limit != 0 || latest != 5 || hours != 24 || next != 123 {
		t.Fatal("migration changed existing policy")
	}
	var saved, owned int
	if e = store.DB.QueryRow("SELECT count(*) FROM playlist_videos WHERE playlist_id='p' AND video_id='v'").Scan(&saved); e != nil || saved != 1 {
		t.Fatal("legacy saved relationship changed", e)
	}
	if e = store.DB.QueryRow("SELECT count(*) FROM youtube_media").Scan(&owned); e != nil || owned != 0 {
		t.Fatal("migration inferred deletion ownership", e)
	}
	if e = migrateYouTubeRetention(store.DB); e != nil {
		t.Fatal("migration not idempotent", e)
	}
}

func TestYouTubePlaylistEndRetentionAndSourceRemoval(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	source := retentionSource(t, a, s, "playlist-order", 0)
	ids := []string{}
	for n := 1; n <= 12; n++ {
		ids = append(ids, retentionVideo(t, a, s, source, n, true))
	}
	// A flat playlist window is oldest-to-newest when additions go at its end.
	a.Store.DB.Exec("UPDATE youtube_sources SET kind='playlist',playlist_order='last' WHERE id=?", source.ID)
	a.Store.DB.Exec("UPDATE youtube_items SET seen_at=1,source_position=CAST(SUBSTR(remote_id,2) AS INTEGER) WHERE source_id=?", source.ID)
	mustStatus(t, call(t, a, s, "PUT", "/api/youtube/"+source.ID, map[string]any{"retention_limit": 10}, nil), 200)
	a.writeMu.Lock()
	e := a.purgeYouTubeLocked()
	a.writeMu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	for index, id := range ids {
		_, e = os.Stat(filepath.Join(a.Config.MediaDir, "originals", id+".mp4"))
		if index < 2 && !os.IsNotExist(e) {
			t.Fatal("kept old playlist entries")
		}
		if index >= 2 && e != nil {
			t.Fatal("removed newer playlist entries", e)
		}
	}
	mustStatus(t, call(t, a, s, "DELETE", "/api/youtube/"+source.ID, nil, nil), 200)
	for _, id := range ids[2:] {
		if _, e = os.Stat(filepath.Join(a.Config.MediaDir, "originals", id+".mp4")); e != nil {
			t.Fatal("source removal erased saved media", e)
		}
	}
	var owned int
	a.Store.DB.QueryRow("SELECT count(*) FROM youtube_media WHERE source_id=?", source.ID).Scan(&owned)
	if owned != 0 {
		t.Fatal("removed source retained purge ownership")
	}
}

func TestYouTubeCleanupRejectsTraversalAndLinkedStorage(t *testing.T) {
	root := t.TempDir()
	id := ID()
	if _, e := youtubeRemovalPaths(root, id, "../outside.mp4"); e == nil {
		t.Fatal("accepted traversal")
	}
	if _, e := youtubeRemovalPaths(root, "../outside", "../outside.mp4"); e == nil {
		t.Fatal("accepted non-video identifier")
	}
	os.MkdirAll(filepath.Join(root, "originals"), 0700)
	os.WriteFile(filepath.Join(root, "originals", id+".mp4"), []byte("keep original"), 0600)
	os.MkdirAll(filepath.Join(root, "streaming"), 0700)
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "keep.txt")
	os.WriteFile(sentinel, []byte("keep"), 0600)
	if e := os.Symlink(outside, filepath.Join(root, "streaming", id)); e != nil {
		t.Skip("symlink privilege not available on this host")
	}
	if _, e := youtubeRemovalPaths(root, id, id+".mp4"); e == nil {
		t.Fatal("accepted linked rendition folder")
	}
	if _, e := os.Stat(sentinel); e != nil {
		t.Fatal("outside target changed")
	}
}
