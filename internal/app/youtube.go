package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Additive migration: source ownership is independent of public My Tube channels.
// Imported media always goes through the normal upload verification/quotas/ACLs.
const youtubeSchema = `
BEGIN;
CREATE TABLE IF NOT EXISTS youtube_sources(
 id TEXT PRIMARY KEY,owner TEXT NOT NULL REFERENCES users(id),url TEXT NOT NULL,
 kind TEXT NOT NULL,title TEXT NOT NULL,playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
 latest INTEGER NOT NULL,playlist_order TEXT NOT NULL DEFAULT 'last',auto_hours INTEGER NOT NULL DEFAULT 0,state TEXT NOT NULL DEFAULT 'queued',
 message TEXT NOT NULL DEFAULT '',created INTEGER NOT NULL,started INTEGER NOT NULL DEFAULT 0,
 finished INTEGER NOT NULL DEFAULT 0,next_check INTEGER NOT NULL DEFAULT 0,UNIQUE(owner,url));
CREATE TABLE IF NOT EXISTS youtube_items(
 source_id TEXT NOT NULL REFERENCES youtube_sources(id) ON DELETE CASCADE,
 remote_id TEXT NOT NULL,title TEXT NOT NULL,state TEXT NOT NULL DEFAULT 'queued',
 message TEXT NOT NULL DEFAULT '',video_id TEXT NOT NULL DEFAULT '',upload_id TEXT NOT NULL DEFAULT '',
 bytes INTEGER NOT NULL DEFAULT 0,total INTEGER NOT NULL DEFAULT 0,updated INTEGER NOT NULL,
 PRIMARY KEY(source_id,remote_id));
INSERT OR IGNORE INTO migrations VALUES(2,0);
COMMIT;`

var youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
var youtubeListID = regexp.MustCompile(`^[A-Za-z0-9_-]{10,150}$`)
var youtubeChannelID = regexp.MustCompile(`^UC[A-Za-z0-9_-]{22}$`)
var youtubeHandle = regexp.MustCompile(`^[\pL\pN_.-]{1,100}$`)
var errYouTubeRateLimited = errors.New("YouTube is limiting requests. Wait before checking this source again")

func youtubeURL(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "@") {
		raw = "https://www.youtube.com/" + raw
	}
	if strings.HasPrefix(raw, "youtube.com/") || strings.HasPrefix(raw, "www.youtube.com/") {
		raw = "https://" + raw
	}
	u, e := url.Parse(raw)
	if e != nil || len(raw) > 500 || u.Scheme != "https" || u.User != nil || u.Port() != "" || (u.Hostname() != "youtube.com" && u.Hostname() != "www.youtube.com" && u.Hostname() != "m.youtube.com") {
		return "", "", errors.New("Paste a YouTube channel URL, @handle or playlist URL")
	}
	if u.Path == "/playlist" || u.Path == "/watch" {
		id := u.Query().Get("list")
		if youtubeListID.MatchString(id) {
			return "https://www.youtube.com/playlist?list=" + id, "playlist", nil
		}
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) > 1 && (parts[len(parts)-1] == "videos" || parts[len(parts)-1] == "featured" || parts[len(parts)-1] == "playlists") {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 1 && strings.HasPrefix(parts[0], "@") && youtubeHandle.MatchString(strings.TrimPrefix(parts[0], "@")) {
		return "https://www.youtube.com/" + url.PathEscape(parts[0]) + "/videos", "channel", nil
	}
	if len(parts) == 2 && ((parts[0] == "channel" && youtubeChannelID.MatchString(parts[1])) || ((parts[0] == "c" || parts[0] == "user") && youtubeHandle.MatchString(parts[1]))) {
		return "https://www.youtube.com/" + parts[0] + "/" + url.PathEscape(parts[1]) + "/videos", "channel", nil
	}
	return "", "", errors.New("Use the channel’s main URL or a playlist URL containing list=")
}

type youtubeSource struct {
	ID            string        `json:"id"`
	Owner         string        `json:"-"`
	URL           string        `json:"url"`
	Kind          string        `json:"kind"`
	Title         string        `json:"title"`
	Playlist      string        `json:"playlist_id"`
	Latest        int           `json:"latest"`
	PlaylistOrder string        `json:"playlist_order"`
	AutoHours     int           `json:"auto_hours"`
	State         string        `json:"state"`
	Message       string        `json:"message"`
	Started       int64         `json:"started"`
	Finished      int64         `json:"finished"`
	Next          int64         `json:"next_check"`
	Items         []youtubeItem `json:"items"`
}
type youtubeItem struct {
	Remote  string `json:"remote_id"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Message string `json:"message"`
	Video   string `json:"video_id"`
	Bytes   int64  `json:"bytes"`
	Total   int64  `json:"total"`
}

const youtubeSourceCols = "id,owner,url,kind,title,playlist_id,latest,playlist_order,auto_hours,state,message,started,finished,next_check"

func scanYouTube(s scanner) (v youtubeSource, e error) {
	e = s.Scan(&v.ID, &v.Owner, &v.URL, &v.Kind, &v.Title, &v.Playlist, &v.Latest, &v.PlaylistOrder, &v.AutoHours, &v.State, &v.Message, &v.Started, &v.Finished, &v.Next)
	return
}
func youtubeBusy(state string) bool {
	return state == "queued" || state == "preparing" || state == "scanning" || state == "downloading" || state == "cancelling"
}
func (a *App) YouTubeList(w http.ResponseWriter, r *http.Request) {
	rows, e := a.Store.DB.Query("SELECT "+youtubeSourceCols+" FROM youtube_sources WHERE owner=? ORDER BY created DESC", who(r).User.ID)
	if e != nil {
		a.failure(w, e)
		return
	}
	out := []youtubeSource{}
	for rows.Next() {
		v, e := scanYouTube(rows)
		if e != nil {
			rows.Close()
			a.failure(w, e)
			return
		}
		out = append(out, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		a.failure(w, e)
		return
	}
	for i := range out {
		out[i].Items = []youtubeItem{}
		rows, e = a.Store.DB.Query("SELECT remote_id,title,state,message,video_id,bytes,total FROM youtube_items WHERE source_id=? ORDER BY updated DESC LIMIT 100", out[i].ID)
		if e != nil {
			a.failure(w, e)
			return
		}
		for rows.Next() {
			var v youtubeItem
			if e = rows.Scan(&v.Remote, &v.Title, &v.State, &v.Message, &v.Video, &v.Bytes, &v.Total); e != nil {
				break
			}
			out[i].Items = append(out[i].Items, v)
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			a.failure(w, e)
			return
		}
	}
	respond(w, 200, out)
}

type youtubeInput struct {
	URL           string `json:"url"`
	Latest        int    `json:"latest"`
	AutoHours     int    `json:"auto_hours"`
	PlaylistOrder string `json:"playlist_order"`
}

func validYouTubeInput(in youtubeInput) bool {
	return in.Latest >= 1 && in.Latest <= 50 && (in.AutoHours == 0 || in.AutoHours == 24) && (in.PlaylistOrder == "first" || in.PlaylistOrder == "last")
}
func (a *App) YouTubeAdd(w http.ResponseWriter, r *http.Request) {
	var in youtubeInput
	if !decode(w, r, &in) {
		return
	}
	if in.PlaylistOrder == "" {
		in.PlaylistOrder = "last"
	}
	u, kind, e := youtubeURL(in.URL)
	if e != nil {
		problem(w, 400, e.Error())
		return
	}
	if !validYouTubeInput(in) {
		problem(w, 400, "Choose 1–50 videos and manual or daily checks")
		return
	}
	uid := who(r).User.ID
	var existing string
	if a.Store.DB.QueryRow("SELECT id FROM youtube_sources WHERE owner=? AND url=?", uid, u).Scan(&existing) == nil {
		problem(w, 409, "This source is already saved. Choose Check for new videos on its card.")
		return
	}
	var count int
	if e = a.Store.DB.QueryRow("SELECT count(*) FROM youtube_sources WHERE owner=?", uid).Scan(&count); e != nil {
		a.failure(w, e)
		return
	}
	if count >= 25 {
		problem(w, 400, "Up to 25 saved YouTube sources per account")
		return
	}
	tx, e := a.Store.DB.Begin()
	if e != nil {
		a.failure(w, e)
		return
	}
	defer tx.Rollback()
	// Make first import usable without a separate channel-creation screen.
	var channel string
	e = tx.QueryRow("SELECT id FROM channels WHERE owner=?", uid).Scan(&channel)
	if e != nil {
		channel = ID()
		_, e = tx.Exec("INSERT INTO channels(id,owner,name,handle,visibility,created)VALUES(?,?,?,?,'private',?)", channel, uid, youtubeText(who(r).User.Name+"’s library", 100), "library-"+channel[:16], now())
		if e != nil {
			a.failure(w, e)
			return
		}
	}
	id, pl := ID(), ID()
	title := "YouTube " + kind
	_, e = tx.Exec("INSERT INTO playlists(id,owner,title,description,visibility,created)VALUES(?,?,?,?,'private',?)", pl, uid, title, u, now())
	if e == nil {
		_, e = tx.Exec("INSERT INTO playlist_search(rowid,title,description) SELECT rowid,title,description FROM playlists WHERE id=?", pl)
	}
	if e == nil {
		_, e = tx.Exec("INSERT INTO youtube_sources(id,owner,url,kind,title,playlist_id,latest,playlist_order,auto_hours,created)VALUES(?,?,?,?,?,?,?,?,?,?)", id, uid, u, kind, title, pl, in.Latest, in.PlaylistOrder, in.AutoHours, now())
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		a.failure(w, e)
		return
	}
	a.Store.Audit(uid, "youtube.add", id)
	respond(w, 202, map[string]string{"id": id})
}
func (a *App) ownedYouTube(w http.ResponseWriter, r *http.Request) (youtubeSource, bool) {
	v, e := scanYouTube(a.Store.DB.QueryRow("SELECT "+youtubeSourceCols+" FROM youtube_sources WHERE id=? AND owner=?", r.PathValue("id"), who(r).User.ID))
	if e != nil {
		problem(w, 404, "YouTube source unavailable")
		return v, false
	}
	return v, true
}
func (a *App) YouTubeSync(w http.ResponseWriter, r *http.Request) {
	v, ok := a.ownedYouTube(w, r)
	if !ok {
		return
	}
	if youtubeBusy(v.State) {
		problem(w, 409, "This import is already in progress")
		return
	}
	_, e := a.Store.DB.Exec("UPDATE youtube_sources SET state='queued',message='Waiting to check YouTube',finished=0 WHERE id=?", v.ID)
	a.failure(w, e)
}
func (a *App) YouTubeSettings(w http.ResponseWriter, r *http.Request) {
	v, ok := a.ownedYouTube(w, r)
	if !ok {
		return
	}
	var in youtubeInput
	if !decode(w, r, &in) {
		return
	}
	if in.PlaylistOrder == "" {
		in.PlaylistOrder = "last"
	}
	if !validYouTubeInput(in) {
		problem(w, 400, "Choose 1–50 videos and manual or daily checks")
		return
	}
	_, e := a.Store.DB.Exec("UPDATE youtube_sources SET latest=?,playlist_order=?,auto_hours=?,next_check=? WHERE id=?", in.Latest, in.PlaylistOrder, in.AutoHours, now()+86400000, v.ID)
	a.failure(w, e)
}
func (a *App) YouTubeCancel(w http.ResponseWriter, r *http.Request) {
	v, ok := a.ownedYouTube(w, r)
	if !ok {
		return
	}
	if !youtubeBusy(v.State) {
		respond(w, 200, map[string]bool{"ok": true})
		return
	}
	a.youtubeMu.Lock()
	active := a.youtubeActive == v.ID && a.youtubeCancel != nil
	state := "cancelled"
	if active {
		state = "cancelling"
		a.youtubeCancel()
	}
	a.youtubeMu.Unlock()
	_, e := a.Store.DB.Exec("UPDATE youtube_sources SET state=?,message='Cancelled; saved videos are kept',finished=?,next_check=? WHERE id=?", state, now(), now()+86400000, v.ID)
	a.failure(w, e)
}
func (a *App) YouTubeRemove(w http.ResponseWriter, r *http.Request) {
	v, ok := a.ownedYouTube(w, r)
	if !ok {
		return
	}
	if youtubeBusy(v.State) {
		problem(w, 409, "Cancel the import before removing its source")
		return
	}
	_, e := a.Store.DB.Exec("DELETE FROM youtube_sources WHERE id=?", v.ID)
	a.failure(w, e)
}

func (a *App) youtubeWorker(ctx context.Context) {
	a.recoverYouTubeImports()
	// A stopped process cannot retain a helper process; restart bounded work.
	a.Store.DB.Exec("UPDATE youtube_sources SET state=CASE WHEN state='cancelling' THEN 'cancelled' ELSE 'queued' END,message='Restarted after interruption' WHERE state IN('preparing','scanning','downloading','cancelling')")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		a.writeMu.Lock()
		v, e := scanYouTube(a.Store.DB.QueryRow("SELECT "+youtubeSourceCols+" FROM youtube_sources WHERE (state='queued' OR (auto_hours>0 AND next_check<=? AND state NOT IN('preparing','scanning','downloading','cancelling'))) AND owner IN(SELECT id FROM users WHERE disabled=0) ORDER BY CASE WHEN state='queued' THEN 0 ELSE 1 END,created LIMIT 1", now()))
		var jobCtx context.Context
		var cancel context.CancelFunc
		if e == nil {
			jobCtx, cancel = context.WithTimeout(ctx, 12*time.Hour)
			a.youtubeMu.Lock()
			a.youtubeActive = v.ID
			a.youtubeCancel = cancel
			a.youtubeMu.Unlock()
			_, e = a.Store.DB.Exec("UPDATE youtube_sources SET state='preparing',message='Preparing verified YouTube tools',started=?,finished=0,next_check=? WHERE id=?", now(), now()+86400000, v.ID)
		}
		a.writeMu.Unlock()
		if e == nil {
			err := a.importYouTube(jobCtx, v)
			a.writeMu.Lock()
			var state string
			a.Store.DB.QueryRow("SELECT state FROM youtube_sources WHERE id=?", v.ID).Scan(&state)
			if ctx.Err() != nil && state != "cancelling" {
				a.Store.DB.Exec("UPDATE youtube_sources SET state='queued',message='Waiting to resume after restart' WHERE id=?", v.ID)
			} else if state == "cancelling" || errors.Is(err, context.Canceled) || errors.Is(jobCtx.Err(), context.Canceled) {
				a.youtubeStatus(v.ID, "cancelled", "Cancelled; saved videos are kept", true)
			} else if err != nil {
				message := err.Error()
				if errors.Is(err, context.DeadlineExceeded) {
					message = "Import timed out. Saved videos are kept; check this source again to retry"
				}
				a.youtubeStatus(v.ID, "failed", message, true)
			}
			if ctx.Err() == nil {
				a.Store.DB.Exec("UPDATE youtube_items SET state='interrupted',message='Not saved; check this source again to retry',updated=? WHERE source_id=? AND state IN('downloading','saving')", now(), v.ID)
			}
			a.youtubeMu.Lock()
			a.youtubeActive = ""
			a.youtubeCancel = nil
			a.youtubeMu.Unlock()
			a.writeMu.Unlock()
			cancel()
			continue
		}
		if cancel != nil {
			cancel()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Recover only uploads recorded by this feature, never ordinary user uploads.
func (a *App) recoverYouTubeImports() {
	rows, e := a.Store.DB.Query("SELECT i.source_id,i.remote_id,i.upload_id,s.owner FROM youtube_items i JOIN youtube_sources s ON s.id=i.source_id WHERE i.upload_id<>''")
	if e != nil {
		return
	}
	type pending struct{ source, remote, upload, owner string }
	list := []pending{}
	for rows.Next() {
		var p pending
		if rows.Scan(&p.source, &p.remote, &p.upload, &p.owner) == nil {
			list = append(list, p)
		}
	}
	rows.Close()
	for _, p := range list {
		if !objectID.MatchString(p.upload) {
			continue
		}
		a.writeMu.Lock()
		var video string
		e = a.Store.DB.QueryRow("SELECT video_id FROM uploads WHERE id=? AND owner=?", p.upload, p.owner).Scan(&video)
		if e == nil && video == "" {
			if err := os.Remove(a.incoming(p.upload)); err == nil || os.IsNotExist(err) {
				a.Store.DB.Exec("DELETE FROM uploads WHERE id=? AND owner=? AND video_id=''", p.upload, p.owner)
				a.Store.DB.Exec("UPDATE youtube_items SET state='interrupted',message='Interrupted download will retry on the next check',upload_id='',bytes=0,total=0 WHERE source_id=? AND remote_id=?", p.source, p.remote)
			}
		}
		if e != nil {
			a.Store.DB.Exec("UPDATE youtube_items SET upload_id='' WHERE source_id=? AND remote_id=?", p.source, p.remote)
		}
		a.writeMu.Unlock()
		if video != "" {
			s, err := scanYouTube(a.Store.DB.QueryRow("SELECT "+youtubeSourceCols+" FROM youtube_sources WHERE id=?", p.source))
			if err == nil {
				a.linkYouTube(s, p.remote, video, "Saved before restart")
			}
		}
	}
}
func (a *App) youtubeStatus(id, state, message string, finished bool) {
	var end int64
	if finished {
		end = now()
	}
	a.Store.DB.Exec("UPDATE youtube_sources SET state=?,message=?,finished=? WHERE id=?", state, message, end, id)
}

type youtubeEntry struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Channel     string         `json:"channel"`
	Uploader    string         `json:"uploader"`
	Live        string         `json:"live_status"`
	IsLive      bool           `json:"is_live"`
	Size        int64          `json:"filesize"`
	Approx      int64          `json:"filesize_approx"`
	Entries     []youtubeEntry `json:"entries"`
}

func (a *App) importYouTube(ctx context.Context, source youtubeSource) error {
	tool, deno, e := a.youtubeTools(ctx, source.ID)
	if e != nil {
		return e
	}
	a.youtubeStatus(source.ID, "scanning", "Reading recent videos from YouTube", false)
	args := youtubeArgs(deno)
	args = append(args, "--flat-playlist", "--dump-single-json", "--skip-download", "--ignore-errors")
	if source.Kind == "playlist" && source.PlaylistOrder == "last" {
		args = append(args, "--playlist-items", fmt.Sprintf("-%d:", source.Latest))
	} else {
		args = append(args, "--playlist-end", strconv.Itoa(source.Latest))
	}
	args = append(args, "--", source.URL)
	output := &boundedYouTubeBuffer{limit: 4 << 20}
	scanCtx, scanCancel := context.WithTimeout(ctx, 5*time.Minute)
	e = a.youtubeCommand(scanCtx, tool, args, output)
	scanCancel()
	if e != nil {
		return e
	}
	var listing youtubeEntry
	if e = json.Unmarshal(output.Bytes(), &listing); e != nil {
		return errors.New("YouTube returned an unreadable channel or playlist")
	}
	if len(listing.Entries) == 0 {
		return errors.New("No videos found. Check that the channel or playlist is public and has available videos")
	}
	title := youtubeText(listing.Title, 180)
	if title == "" {
		title = source.Title
	}
	a.writeMu.Lock()
	_, e = a.Store.DB.Exec("UPDATE youtube_sources SET title=? WHERE id=?", title, source.ID)
	// Preserve the source name on its private collection and in playlist search.
	if e == nil {
		_, e = a.Store.DB.Exec("UPDATE playlists SET title=? WHERE id=? AND owner=?", title, source.Playlist, source.Owner)
	}
	if e == nil {
		_, e = a.Store.DB.Exec("DELETE FROM playlist_search WHERE rowid IN(SELECT rowid FROM playlists WHERE id=?)", source.Playlist)
	}
	if e == nil {
		_, e = a.Store.DB.Exec("INSERT INTO playlist_search(rowid,title,description) SELECT rowid,title,description FROM playlists WHERE id=?", source.Playlist)
	}
	a.writeMu.Unlock()
	if e != nil {
		return e
	}
	failures, saved, skipped := 0, 0, 0
	seen := map[string]bool{}
	for i, entry := range listing.Entries {
		if i >= source.Latest {
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !youtubeID.MatchString(entry.ID) || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		entry.Title = youtubeText(entry.Title, 195)
		if entry.Title == "" {
			entry.Title = entry.ID
		}
		_, e = a.Store.DB.Exec("INSERT INTO youtube_items(source_id,remote_id,title,updated)VALUES(?,?,?,?) ON CONFLICT(source_id,remote_id) DO UPDATE SET title=excluded.title,updated=excluded.updated", source.ID, entry.ID, entry.Title, now())
		if e != nil {
			return e
		}
		// Deduplicate across every saved source belonging to the same user.
		var video string
		a.Store.DB.QueryRow("SELECT i.video_id FROM youtube_items i JOIN youtube_sources s ON s.id=i.source_id JOIN videos v ON v.id=i.video_id WHERE s.owner=? AND i.remote_id=? AND v.deleted=0 LIMIT 1", source.Owner, entry.ID).Scan(&video)
		if video != "" {
			if e = a.linkYouTube(source, entry.ID, video, "Already in your library"); e != nil {
				return e
			}
			skipped++
			continue
		}
		if entry.IsLive || entry.Live == "is_live" || entry.Live == "is_upcoming" {
			a.youtubeItemState(source.ID, entry.ID, "failed", "Live or upcoming videos can be imported after they finish")
			failures++
			continue
		}
		a.youtubeStatus(source.ID, "downloading", fmt.Sprintf("Video %d of %d · %s", i+1, min(len(listing.Entries), source.Latest), entry.Title), false)
		e = a.downloadYouTube(ctx, tool, deno, source, entry)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e != nil {
			message := e.Error()
			if errors.Is(e, context.DeadlineExceeded) {
				message = "Video download timed out; check this source again to retry"
			}
			a.youtubeItemState(source.ID, entry.ID, "failed", message)
			failures++
			if errors.Is(e, errYouTubeRateLimited) {
				return e
			}
		} else {
			saved++
		}
	}
	if len(seen) == 0 {
		return errors.New("This source has no available YouTube videos")
	}
	state := "completed"
	if failures > 0 {
		state = "partial"
	}
	a.youtubeStatus(source.ID, state, fmt.Sprintf("Finished · %d saved · %d already in your library · %d need attention", saved, skipped, failures), true)
	return nil
}
func youtubeText(s string, n int) string {
	s = strings.ToValidUTF8(strings.TrimSpace(s), "")
	if len(s) > n {
		s = s[:n]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
	}
	return s
}
func youtubeArgs(deno string) []string {
	return []string{"--ignore-config", "--no-plugin-dirs", "--no-remote-components", "--no-js-runtimes", "--js-runtimes", "deno:" + deno, "--no-cache-dir", "--no-color", "--socket-timeout", "20", "--retries", "2", "--fragment-retries", "2", "--extractor-retries", "2", "--no-warnings"}
}
func (a *App) youtubeCommand(ctx context.Context, tool string, args []string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, tool, args...)
	root, e := filepath.Abs(filepath.Join(a.Config.DataDir, "helpers"))
	if e != nil {
		return e
	}
	if e := safeYouTubeDir(root); e != nil {
		return e
	}
	stage, e := os.MkdirTemp(root, ".youtube-run-")
	if e != nil {
		return e
	}
	// MkdirTemp created this exact child. RemoveAll never follows child symlinks.
	if filepath.Dir(stage) != root {
		return errors.New("Unexpected YouTube staging path")
	}
	defer os.RemoveAll(stage)
	cmd.Env = youtubeEnvironment(stage)
	cmd.Dir = stage
	cmd.Stdout = out
	stderr := &youtubeErrorBuffer{}
	cmd.Stderr = stderr
	cmd.WaitDelay = 5 * time.Second
	if e := runYouTubeLimited(cmd); e != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		text := strings.ToLower(stderr.String())
		switch {
		case strings.Contains(text, "429") || strings.Contains(text, "rate limit"):
			return errYouTubeRateLimited
		case strings.Contains(text, "sign in") || strings.Contains(text, "sign-in") || strings.Contains(text, "private video") || strings.Contains(text, "members-only"):
			return errors.New("YouTube requires sign-in for this video or source; only publicly downloadable videos can be imported")
		case strings.Contains(text, "unavailable") || strings.Contains(text, "removed"):
			return errors.New("This YouTube video or source is unavailable")
		case strings.Contains(text, "format is not available"):
			return errors.New("YouTube did not offer a supported MP4 download for this video")
		default:
			return errors.New("YouTube import failed. Check your connection and retry; saved videos are kept")
		}
	}
	return nil
}

type boundedYouTubeBuffer struct {
	buffer bytes.Buffer
	limit  int
}
type youtubeErrorBuffer struct{ tail []byte }

func (b *youtubeErrorBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if len(p) >= 8192 {
		b.tail = append(b.tail[:0], p[len(p)-8192:]...)
	} else {
		b.tail = append(b.tail, p...)
		if len(b.tail) > 8192 {
			b.tail = append([]byte{}, b.tail[len(b.tail)-8192:]...)
		}
	}
	return n, nil
}
func (b *youtubeErrorBuffer) String() string { return string(b.tail) }

func (b *boundedYouTubeBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *boundedYouTubeBuffer) String() string { return b.buffer.String() }

func (b *boundedYouTubeBuffer) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > b.limit {
		return 0, errors.New("YouTube response exceeded its limit")
	}
	return b.buffer.Write(p)
}
func (a *App) youtubeItemState(source, remote, state, message string) {
	a.Store.DB.Exec("UPDATE youtube_items SET state=?,message=?,updated=? WHERE source_id=? AND remote_id=?", state, youtubeText(message, 500), now(), source, remote)
}
func (a *App) linkYouTube(s youtubeSource, remote, video, message string) error {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	tx, e := a.Store.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec("INSERT OR IGNORE INTO playlist_videos(playlist_id,video_id,position) SELECT ?,?,COALESCE(MAX(position),0)+1 FROM playlist_videos WHERE playlist_id=?", s.Playlist, video, s.Playlist)
	if e == nil {
		_, e = tx.Exec("UPDATE youtube_items SET video_id=?,upload_id='',state='saved',message=?,updated=? WHERE source_id=? AND remote_id=?", video, message, now(), s.ID, remote)
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}

func (a *App) downloadYouTube(parent context.Context, tool, deno string, s youtubeSource, entry youtubeEntry) error {
	ctx, cancel := context.WithTimeout(parent, time.Duration(a.Config.JobMinutes)*time.Minute)
	defer cancel()
	watch := "https://www.youtube.com/watch?v=" + entry.ID
	args := youtubeArgs(deno)
	// Prefer separate high-quality streams when verified media tools exist;
	// otherwise use a single MP4. Both paths publish through ordinary uploads.
	format := "best[ext=mp4][vcodec^=avc1][acodec^=mp4a][height<=720][protocol=https]/best[ext=mp4][vcodec^=avc1][acodec^=mp4a][height<=720][protocol=http]"
	ffmpeg := a.helper(a.Config.FFmpeg, "ffmpeg")
	if ffmpeg == "" && a.Config.AutoHelpers {
		a.youtubeStatus(s.ID, "preparing", "Preparing video tools for YouTube downloads", false)
		if a.prepareMedia(ctx) == nil {
			ffmpeg = a.helper(a.Config.FFmpeg, "ffmpeg")
		}
		a.youtubeStatus(s.ID, "downloading", "Downloading · "+entry.Title, false)
	}
	adaptive := ffmpeg != "" && a.helper(a.Config.FFprobe, "ffprobe") != ""
	if adaptive {
		format = "bestvideo[ext=mp4][vcodec^=avc1][height<=720][protocol=https]+bestaudio[ext=m4a][protocol=https]/" + format
	}
	args = append(args, "--no-playlist", "--skip-download", "--dump-json", "--format", format, "--", watch)
	metadata := &boundedYouTubeBuffer{limit: 2 << 20}
	if e := a.youtubeCommand(ctx, tool, args, metadata); e != nil {
		return e
	}
	var detail youtubeEntry
	if e := json.Unmarshal(metadata.Bytes(), &detail); e != nil {
		return errors.New("YouTube video metadata is unreadable")
	}
	if detail.ID != entry.ID || detail.IsLive || detail.Live == "is_live" || detail.Live == "is_upcoming" {
		return errors.New("Live or mismatched videos cannot be imported")
	}
	title := youtubeText(detail.Title, 195)
	if title == "" {
		title = entry.Title
	}
	// Reserve an upper bound when YouTube does not provide an exact size.
	bound := detail.Size
	if adaptive {
		bound = detail.Approx
		if bound > 0 {
			bound = bound + bound/10 + (1 << 20)
		}
	}
	if bound <= 0 {
		bound = a.Config.UploadMB << 20
	}
	a.writeMu.Lock()
	var disabled bool
	e := a.Store.DB.QueryRow("SELECT disabled FROM users WHERE id=?", s.Owner).Scan(&disabled)
	if e != nil || disabled {
		a.writeMu.Unlock()
		return errors.New("Your My Tube account is unavailable")
	}
	u, e := a.begin(s.Owner, entry.ID+".mp4", title, bound, "private")
	if e == nil {
		_, e = a.Store.DB.Exec("UPDATE youtube_items SET state='downloading',message='Downloading video',upload_id=?,bytes=0,total=?,updated=? WHERE source_id=? AND remote_id=?", u.ID, detail.Size, now(), s.ID, entry.ID)
	}
	a.writeMu.Unlock()
	if e != nil {
		if u.ID != "" {
			a.writeMu.Lock()
			if err := os.Remove(a.incoming(u.ID)); err == nil || os.IsNotExist(err) {
				a.Store.DB.Exec("DELETE FROM uploads WHERE id=? AND owner=? AND video_id=''", u.ID, s.Owner)
			}
			a.writeMu.Unlock()
		}
		return e
	}
	defer func() {
		// Only the reserved incomplete upload is removed. Published originals stay.
		a.writeMu.Lock()
		defer a.writeMu.Unlock()
		var video string
		if a.Store.DB.QueryRow("SELECT video_id FROM uploads WHERE id=?", u.ID).Scan(&video) == nil && video == "" {
			if err := os.Remove(a.incoming(u.ID)); err == nil || os.IsNotExist(err) {
				a.Store.DB.Exec("DELETE FROM uploads WHERE id=? AND video_id=''", u.ID)
			}
		}
	}()
	f, e := os.OpenFile(a.incoming(u.ID), os.O_WRONLY|os.O_TRUNC, 0600)
	if e != nil {
		return errors.New("Cannot open video staging storage")
	}
	writer := &youtubeDownloadWriter{file: f, limit: bound, update: func(n int64) {
		a.Store.DB.Exec("UPDATE youtube_items SET bytes=?,updated=? WHERE source_id=? AND remote_id=?", n, now(), s.ID, entry.ID)
	}}
	args = youtubeArgs(deno)
	if adaptive {
		// Stream a fragmented MP4 directly into our quota-limited writer. This
		// avoids complete audio/video/merged staging copies on the host disk.
		args = append(args, "--ffmpeg-location", ffmpeg, "--downloader", "ffmpeg", "--downloader-args", "ffmpeg_o:-f mp4 -movflags +frag_keyframe+empty_moov+default_base_moof -loglevel error -nostats", "--downloader-args", "ffmpeg_i:-rw_timeout 20000000")
	}
	args = append(args, "--no-playlist", "--no-progress", "--no-part", "--no-continue", "--fixup", "never", "--hls-prefer-native", "--format", format, "--max-filesize", strconv.FormatInt(bound, 10), "--output", "-", "--", watch)
	e = a.youtubeCommand(ctx, tool, args, writer)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if writer.exceeded {
		return errors.New("Video exceeds the reserved upload size; no partial video was saved")
	}
	if e != nil {
		return e
	}
	if writer.n < 1 {
		return errors.New("YouTube supplied no video bytes")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	a.youtubeItemState(s.ID, entry.ID, "saving", "Download finished; verifying and saving video")
	a.writeMu.Lock()
	u.Size = writer.n
	u.Offset = writer.n
	_, e = a.Store.DB.Exec("UPDATE uploads SET size=?,offset=? WHERE id=?", u.Size, u.Offset, u.ID)
	var video string
	// A restart after publication but before linking is also caught by this hash.
	if e == nil {
		file, err := os.Open(a.incoming(u.ID))
		if err != nil {
			e = err
		} else {
			h := sha256.New()
			_, e = io.Copy(h, file)
			file.Close()
			if e == nil {
				a.Store.DB.QueryRow("SELECT id FROM videos WHERE owner=? AND sha256=? AND deleted=0", s.Owner, hex.EncodeToString(h.Sum(nil))).Scan(&video)
			}
		}
	}
	if e == nil && video == "" {
		r, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost/", nil)
		video, e = a.finish(r, u)
		if e == nil {
			creator := youtubeText(detail.Channel, 200)
			if creator == "" {
				creator = youtubeText(detail.Uploader, 200)
			}
			description := youtubeText(detail.Description, 9000) + "\n\nImported from YouTube: " + watch + "\nOriginal channel: " + creator
			_, e = a.Store.DB.Exec("UPDATE videos SET description=? WHERE id=?", description, video)
			a.indexVideo(video)
		}
	}
	a.writeMu.Unlock()
	if e != nil {
		return e
	}
	a.Store.DB.Exec("UPDATE youtube_items SET bytes=?,total=? WHERE source_id=? AND remote_id=?", writer.n, writer.n, s.ID, entry.ID)
	return a.linkYouTube(s, entry.ID, video, "Saved to your private library")
}

type youtubeDownloadWriter struct {
	file     *os.File
	n, limit int64
	last     time.Time
	update   func(int64)
	exceeded bool
}

func (w *youtubeDownloadWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.limit-w.n {
		w.exceeded = true
		return 0, errors.New("Video exceeds upload limit")
	}
	n, e := w.file.Write(p)
	w.n += int64(n)
	if time.Since(w.last) > time.Second {
		w.update(w.n)
		w.last = time.Now()
	}
	return n, e
}

// Keep import staging inside the app's private media area, never the host TEMP.
func youtubeEnvironment(dir string) []string {
	env := []string{}
	for _, v := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(v, "=", 2)[0])
		if strings.HasPrefix(key, "YT_DLP") || strings.HasPrefix(key, "DENO_") || key == "TEMP" || key == "TMP" || key == "TMPDIR" {
			continue
		}
		env = append(env, v)
	}
	return append(env, "TEMP="+dir, "TMP="+dir, "TMPDIR="+dir, "DENO_DIR="+filepath.Join(dir, "deno-cache"), "DENO_NO_UPDATE_CHECK=1")
}
