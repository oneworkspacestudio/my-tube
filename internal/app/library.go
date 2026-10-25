package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Video struct {
	ID             string  `json:"id"`
	Owner          string  `json:"owner"`
	ChannelID      string  `json:"channel_id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Tags           string  `json:"tags"`
	Category       string  `json:"category"`
	Visibility     string  `json:"visibility"`
	Original       string  `json:"-"`
	Filename       string  `json:"filename"`
	Size           int64   `json:"size"`
	SHA256         string  `json:"-"`
	Duration       float64 `json:"duration"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Direct         bool    `json:"direct"`
	Status         string  `json:"status"`
	Error          string  `json:"error"`
	Created        int64   `json:"created"`
	Views          int64   `json:"views"`
	WatchSeconds   float64 `json:"watch_seconds"`
	Chapters       string  `json:"chapters"`
	Channel        string  `json:"channel"`
	Handle         string  `json:"handle"`
	Likes          int     `json:"likes"`
	Liked          bool    `json:"liked"`
	Later          bool    `json:"later"`
	Subscribed     bool    `json:"subscribed"`
	Position       float64 `json:"position"`
	Recommendation string  `json:"recommendation"`
	HLS            bool    `json:"hls"`
}

const videoCols = `v.id,v.owner,v.channel_id,v.title,v.description,v.tags,v.category,v.visibility,v.original,v.filename,v.size,v.sha256,v.duration,v.width,v.height,v.direct,v.status,v.error,v.created,v.views,v.watch_seconds,v.chapters,c.name,c.handle`

func scanVideo(s scanner) (Video, error) {
	var v Video
	e := s.Scan(&v.ID, &v.Owner, &v.ChannelID, &v.Title, &v.Description, &v.Tags, &v.Category, &v.Visibility, &v.Original, &v.Filename, &v.Size, &v.SHA256, &v.Duration, &v.Width, &v.Height, &v.Direct, &v.Status, &v.Error, &v.Created, &v.Views, &v.WatchSeconds, &v.Chapters, &v.Channel, &v.Handle)
	return v, e
}

// One predicate is used for metadata, discovery, originals and every derivative.
func (a *App) can(v Video, uid string, discovery bool) bool {
	if uid != "" && v.Owner == uid {
		return true
	}
	if v.Visibility == "private" {
		if uid == "" {
			return false
		}
		var n int
		a.Store.DB.QueryRow("SELECT count(*) FROM video_access WHERE video_id=? AND user_id=?", v.ID, uid).Scan(&n)
		return n > 0
	}
	if v.Visibility == "unlisted" {
		return !discovery && (uid != "" || a.Config.PublicAccess)
	}
	return uid != "" || a.Config.PublicAccess
}
func (a *App) video(id, uid string) (Video, error) {
	v, e := scanVideo(a.Store.DB.QueryRow("SELECT "+videoCols+" FROM videos v JOIN channels c ON c.id=v.channel_id WHERE v.id=? AND v.deleted=0", id))
	if e != nil || !objectID.MatchString(v.ID) || !originalKey.MatchString(v.Original) || !a.can(v, uid, false) {
		return Video{}, sql.ErrNoRows
	}
	return v, nil
}
func (a *App) enrich(v *Video, uid string) {
	// Source channels are private collections, never a second identity/account.
	// Preserve the original video ACL and expose source membership only to owner.
	if uid != "" && v.Owner == uid {
		var id, name, handle string
		if a.Store.DB.QueryRow(`SELECT s.id,c.name,c.handle FROM youtube_items i
   JOIN youtube_sources s ON s.id=i.source_id JOIN youtube_channels c ON c.source_id=s.id
   WHERE i.video_id=? AND s.owner=? ORDER BY s.created,s.id LIMIT 1`, v.ID, uid).Scan(&id, &name, &handle) == nil {
			v.ChannelID, v.Channel, v.Handle = id, name, handle
		}
	}
	a.Store.DB.QueryRow("SELECT count(*) FROM likes WHERE video_id=?", v.ID).Scan(&v.Likes)
	if uid != "" {
		a.Store.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM likes WHERE user_id=? AND video_id=?),EXISTS(SELECT 1 FROM later WHERE user_id=? AND video_id=?),EXISTS(SELECT 1 FROM subscriptions WHERE user_id=? AND channel_id=?)", uid, v.ID, uid, v.ID, uid, v.ChannelID).Scan(&v.Liked, &v.Later, &v.Subscribed)
		a.Store.DB.QueryRow("SELECT position FROM history WHERE user_id=? AND video_id=?", uid, v.ID).Scan(&v.Position)
	}
	_, e := os.Stat(filepath.Join(a.Config.MediaDir, "streaming", v.ID, "master.m3u8"))
	v.HLS = e == nil
}
func (a *App) ListVideos(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	q := r.URL.Query()
	where := "v.deleted=0"
	args := []any{}
	search := strings.TrimSpace(q.Get("q"))
	if search != "" {
		if len(search) > 200 {
			problem(w, 400, "Search too long")
			return
		}
		where += " AND v.rowid IN(SELECT rowid FROM video_search WHERE video_search MATCH ?)"
		args = append(args, `"`+strings.ReplaceAll(search, `"`, `""`)+`"`)
	}
	if ch := q.Get("channel"); ch != "" {
		where += " AND (v.channel_id=? OR EXISTS(SELECT 1 FROM youtube_items yi JOIN youtube_sources ys ON ys.id=yi.source_id WHERE yi.video_id=v.id AND ys.id=? AND ys.owner=? AND ys.kind='channel'))"
		args = append(args, ch, ch, uid)
	}
	if cat := q.Get("category"); cat != "" && cat != "All" {
		where += " AND v.category=?"
		args = append(args, cat)
	}
	if d, _ := strconv.Atoi(q.Get("days")); d > 0 {
		where += " AND v.created>?"
		args = append(args, now()-int64(d)*86400000)
	}
	switch q.Get("duration") {
	case "short":
		where += " AND v.duration<240"
	case "long":
		where += " AND v.duration>=1200"
	}
	view := q.Get("view")
	switch view {
	case "studio":
		where += " AND v.owner=?"
		args = append(args, uid)
	case "subscriptions":
		where += " AND EXISTS(SELECT 1 FROM subscriptions s WHERE s.channel_id=v.channel_id AND s.user_id=?)"
		args = append(args, uid)
	case "history", "later", "likes":
		table := map[string]string{"history": "history", "later": "later", "likes": "likes"}[view]
		where += " AND EXISTS(SELECT 1 FROM " + table + " t WHERE t.video_id=v.id AND t.user_id=?)"
		args = append(args, uid)
	case "shorts":
		where += " AND v.duration<=180 AND v.height>v.width"
	}
	if pl := q.Get("playlist"); pl != "" {
		if !a.canPlaylist(pl, uid) {
			problem(w, 404, "Playlist unavailable")
			return
		}
		where += " AND EXISTS(SELECT 1 FROM playlist_videos p WHERE p.video_id=v.id AND p.playlist_id=?)"
		args = append(args, pl)
	}
	order := "v.created DESC"
	if q.Get("sort") == "popular" {
		order = "v.views DESC,v.created DESC"
	}
	if q.Get("sort") == "oldest" {
		order = "v.created ASC"
	}
	rows, e := a.Store.DB.Query("SELECT "+videoCols+" FROM videos v JOIN channels c ON c.id=v.channel_id WHERE "+where+" ORDER BY "+order+" LIMIT 500", args...)
	if e != nil {
		problem(w, 500, "Search unavailable")
		return
	}
	all := []Video{}
	for rows.Next() {
		v, e := scanVideo(rows)
		if e == nil {
			all = append(all, v)
		}
	}
	rows.Close()
	items := []Video{}
	personalize := false
	if uid != "" {
		a.Store.DB.QueryRow("SELECT personalize FROM users WHERE id=?", uid).Scan(&personalize)
	}
	interests := map[string]bool{}
	if personalize {
		rows, _ := a.Store.DB.Query("SELECT DISTINCT v.category FROM history h JOIN videos v ON v.id=h.video_id WHERE h.user_id=? AND h.position>v.duration*.5 ORDER BY h.updated DESC LIMIT 10", uid)
		if rows != nil {
			for rows.Next() {
				var cat string
				rows.Scan(&cat)
				interests[cat] = true
			}
			rows.Close()
		}
	}
	for _, v := range all {
		if !a.can(v, uid, view != "studio" && q.Get("playlist") == "" && view != "history" && view != "later" && view != "likes") {
			continue
		}
		a.enrich(&v, uid)
		v.Recommendation = "Recently uploaded"
		if personalize && v.Subscribed {
			v.Recommendation = "From a channel you follow"
		} else if personalize && interests[v.Category] {
			v.Recommendation = "More in a category you watch"
		} else if v.Views > 10 {
			v.Recommendation = "Popular in this library"
		}
		items = append(items, v)
	}
	if personalize && q.Get("recommend") == "1" {
		score := func(v Video) int {
			n := 0
			if v.Subscribed {
				n += 4
			}
			if interests[v.Category] {
				n += 2
			}
			if v.Position > v.Duration*.9 {
				n -= 6
			}
			return n
		}
		sort.SliceStable(items, func(i, j int) bool { return score(items[i]) > score(items[j]) })
	}
	respond(w, 200, items)
}
func (a *App) GetVideo(w http.ResponseWriter, r *http.Request) {
	v, e := a.video(r.PathValue("id"), who(r).User.ID)
	if e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	a.enrich(&v, who(r).User.ID)
	respond(w, 200, v)
}
func visibility(s string) bool { return s == "public" || s == "unlisted" || s == "private" }
func (a *App) indexVideo(id string) error {
	_, e := a.Store.DB.Exec(`DELETE FROM video_search WHERE rowid=(SELECT rowid FROM videos WHERE id=?);INSERT INTO video_search(rowid,title,description,tags,channel,category)SELECT v.rowid,v.title,v.description,v.tags,c.name,v.category FROM videos v JOIN channels c ON c.id=v.channel_id WHERE v.id=?`, id, id)
	return e
}
func (a *App) EditVideo(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	v, e := a.video(r.PathValue("id"), uid)
	if e != nil || v.Owner != uid {
		problem(w, 404, "Video unavailable")
		return
	}
	var in struct {
		Title, Description, Tags, Category, Visibility string
		Chapters                                       []struct {
			Time  float64 `json:"time"`
			Title string  `json:"title"`
		}
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Title) == "" || len(in.Title) > 200 || len(in.Description) > 20000 || len(in.Tags) > 2000 || len(in.Category) > 80 || !visibility(in.Visibility) || in.Visibility == "public" && !a.Config.PublicUploads {
		problem(w, 400, "Invalid metadata or public uploads disabled")
		return
	}
	last := -1.0
	for _, c := range in.Chapters {
		if c.Time < 0 || c.Time <= last || c.Time > v.Duration || len(c.Title) > 200 {
			problem(w, 400, "Chapters must increase within video duration")
			return
		}
		last = c.Time
	}
	chapters, _ := json.Marshal(in.Chapters)
	_, e = a.Store.DB.Exec("UPDATE videos SET title=?,description=?,tags=?,category=?,visibility=?,chapters=? WHERE id=?", in.Title, in.Description, in.Tags, in.Category, in.Visibility, string(chapters), v.ID)
	if e != nil {
		problem(w, 500, "Could not save video")
		return
	}
	a.indexVideo(v.ID)
	a.Store.Audit(uid, "video.edit", v.ID)
	a.GetVideo(w, r)
}
func (a *App) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	v, e := a.video(r.PathValue("id"), uid)
	if e != nil || v.Owner != uid {
		problem(w, 404, "Video unavailable")
		return
	}
	_, e = a.Store.DB.Exec("UPDATE videos SET deleted=1,visibility='private' WHERE id=?", v.ID)
	if e != nil {
		problem(w, 500, "Delete failed")
		return
	}
	a.Store.Audit(uid, "video.delete", v.ID)
	respond(w, 200, map[string]any{"ok": true, "retention": "Original retained until administrator storage reconciliation"})
}

// ReplaceVideo atomically publishes a verified new upload and retires the old
// record. It deliberately has a new URL; the previous original is recoverable.
func (a *App) ReplaceVideo(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	old, e := a.video(r.PathValue("id"), uid)
	if e != nil || old.Owner != uid {
		problem(w, 404, "Owned video unavailable")
		return
	}
	var in struct {
		ReplacementID string `json:"replacement_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	next, e := a.video(in.ReplacementID, uid)
	if e != nil || next.Owner != uid || next.ID == old.ID || next.Visibility != "private" {
		problem(w, 400, "Choose a newly uploaded private video that you own")
		return
	}
	tx, e := a.Store.DB.Begin()
	if e != nil {
		a.failure(w, e)
		return
	}
	defer tx.Rollback()
	_, e = tx.Exec("UPDATE videos SET title=?,description=?,tags=?,category=?,visibility=? WHERE id=?", old.Title, old.Description, old.Tags, old.Category, old.Visibility, next.ID)
	if e == nil {
		_, e = tx.Exec("UPDATE videos SET deleted=1,visibility='private' WHERE id=?", old.ID)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		a.failure(w, e)
		return
	}
	a.indexVideo(next.ID)
	a.Store.Audit(uid, "video.replace", old.ID+":"+next.ID)
	respond(w, 200, map[string]string{"id": next.ID, "previous_id": old.ID})
}

func (a *App) VideoAccess(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	v, e := a.video(r.PathValue("id"), uid)
	if e != nil || v.Owner != uid {
		problem(w, 404, "Video unavailable")
		return
	}
	var in struct {
		Subject string `json:"subject"`
		Allow   bool   `json:"allow"`
	}
	if !decode(w, r, &in) {
		return
	}
	var user string
	e = a.Store.DB.QueryRow("SELECT id FROM users WHERE issuer=? AND subject=?", a.Config.Issuer, in.Subject).Scan(&user)
	if e != nil {
		problem(w, 404, "Account must sign in first")
		return
	}
	if in.Allow {
		_, e = a.Store.DB.Exec("INSERT OR IGNORE INTO video_access VALUES(?,?)", v.ID, user)
	} else {
		_, e = a.Store.DB.Exec("DELETE FROM video_access WHERE video_id=? AND user_id=?", v.ID, user)
	}
	if e != nil {
		problem(w, 500, "Access update failed")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}

type Channel struct {
	ID            string        `json:"id"`
	Owner         string        `json:"owner"`
	Name          string        `json:"name"`
	Handle        string        `json:"handle"`
	Description   string        `json:"description"`
	Avatar        string        `json:"avatar"`
	Banner        string        `json:"banner"`
	Visibility    string        `json:"visibility"`
	Subscribers   int           `json:"subscribers"`
	Subscribed    bool          `json:"subscribed"`
	Imported      bool          `json:"imported"`
	SourceURL     string        `json:"source_url,omitempty"`
	Playlist      string        `json:"playlist_id,omitempty"`
	ImportState   string        `json:"import_state,omitempty"`
	ImportMessage string        `json:"import_message,omitempty"`
	Items         []youtubeItem `json:"items,omitempty"`
}

const channelCols = "c.id,c.owner,c.name,c.handle,c.description,c.avatar,c.banner,c.visibility,(SELECT count(*) FROM subscriptions WHERE channel_id=c.id)"

func scanChannel(r scanner) (Channel, error) {
	var c Channel
	e := r.Scan(&c.ID, &c.Owner, &c.Name, &c.Handle, &c.Description, &c.Avatar, &c.Banner, &c.Visibility, &c.Subscribers)
	return c, e
}
func (a *App) Channels(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	rows, e := a.Store.DB.Query("SELECT "+channelCols+" FROM channels c WHERE (visibility='public' AND (?<>'' OR ?)) OR owner=? ORDER BY name", uid, a.Config.PublicAccess, uid)
	if e != nil {
		problem(w, 500, "Channels unavailable")
		return
	}
	defer rows.Close()
	out := []Channel{}
	for rows.Next() {
		c, e := scanChannel(rows)
		if e == nil {
			out = append(out, c)
		}
	}
	rows.Close()
	imported, e := a.youtubeChannels(uid)
	if e != nil {
		a.failure(w, e)
		return
	}
	out = append(out, imported...)
	respond(w, 200, out)
}
func (a *App) Channel(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	c, e := scanChannel(a.Store.DB.QueryRow("SELECT "+channelCols+" FROM channels c WHERE id=? OR handle=?", r.PathValue("id"), strings.TrimPrefix(r.PathValue("id"), "@")))
	if e != nil {
		c, e = a.youtubeChannel(r.PathValue("id"), uid)
	}
	if e != nil || (c.Owner != uid && (c.Visibility == "private" || uid == "" && !a.Config.PublicAccess)) {
		problem(w, 404, "Channel unavailable")
		return
	}
	a.Store.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM subscriptions WHERE user_id=? AND channel_id=?)", uid, c.ID).Scan(&c.Subscribed)
	respond(w, 200, c)
}

var handlePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{2,39}$`)

func (a *App) SaveChannel(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Handle, Description, Visibility string }
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Name)) == 0 || len(in.Name) > 100 || !handlePattern.MatchString(in.Handle) || len(in.Description) > 5000 || in.Visibility != "public" && in.Visibility != "private" {
		problem(w, 400, "Use a name and a unique handle of 3–40 letters, digits, hyphens or underscores")
		return
	}
	uid := who(r).User.ID
	id := ID()
	_, e := a.Store.DB.Exec(`INSERT INTO channels(id,owner,name,handle,description,visibility,created)VALUES(?,?,?,?,?,?,?) ON CONFLICT(owner)DO UPDATE SET name=excluded.name,handle=excluded.handle,description=excluded.description,visibility=excluded.visibility`, id, uid, in.Name, in.Handle, in.Description, in.Visibility, now())
	if e != nil {
		problem(w, 409, "Handle is already used")
		return
	}
	a.Store.DB.QueryRow("SELECT id FROM channels WHERE owner=?", uid).Scan(&id)
	rows, _ := a.Store.DB.Query("SELECT id FROM videos WHERE channel_id=?", id)
	ids := []string{}
	if rows != nil {
		for rows.Next() {
			var x string
			rows.Scan(&x)
			ids = append(ids, x)
		}
		rows.Close()
	}
	for _, x := range ids {
		a.indexVideo(x)
	}
	a.Store.Audit(uid, "channel.save", id)
	respond(w, 200, map[string]string{"id": id})
}
func (a *App) ownerChannel(uid string) (string, error) {
	var id string
	e := a.Store.DB.QueryRow("SELECT id FROM channels WHERE owner=?", uid).Scan(&id)
	return id, e
}
func (a *App) failure(w http.ResponseWriter, e error) {
	if e != nil {
		a.Logger.Printf(`{"event":"database_error","detail":%q}`, fmt.Sprint(e))
		problem(w, 500, "Database operation failed")
	} else {
		respond(w, 200, map[string]bool{"ok": true})
	}
}
