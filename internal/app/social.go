package app

import (
	"database/sql"

	"net/http"
	"strings"
	"time"
)

func (a *App) toggle(w http.ResponseWriter, r *http.Request, table string) {
	uid := who(r).User.ID
	v, e := a.video(r.PathValue("id"), uid)
	if e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Enabled {
		_, e = a.Store.DB.Exec("INSERT OR IGNORE INTO "+table+" VALUES(?,?)", uid, v.ID)
	} else {
		_, e = a.Store.DB.Exec("DELETE FROM "+table+" WHERE user_id=? AND video_id=?", uid, v.ID)
	}
	a.failure(w, e)
}
func (a *App) Like(w http.ResponseWriter, r *http.Request)  { a.toggle(w, r, "likes") }
func (a *App) Later(w http.ResponseWriter, r *http.Request) { a.toggle(w, r, "later") }
func (a *App) Subscribe(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	var owner, vis string
	ch := r.PathValue("id")
	e := a.Store.DB.QueryRow("SELECT owner,visibility FROM channels WHERE id=?", ch).Scan(&owner, &vis)
	if e != nil || vis == "private" && owner != uid {
		problem(w, 404, "Channel unavailable")
		return
	}
	if owner == uid {
		problem(w, 400, "You own this channel")
		return
	}
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !decode(w, r, &in) {
		return
	}
	var result sql.Result
	if in.Enabled {
		result, e = a.Store.DB.Exec("INSERT OR IGNORE INTO subscriptions VALUES(?,?,?)", uid, ch, now())
	} else {
		result, e = a.Store.DB.Exec("DELETE FROM subscriptions WHERE user_id=? AND channel_id=?", uid, ch)
	}
	if e == nil {
		n, _ := result.RowsAffected()
		if n > 0 {
			delta := 1
			if !in.Enabled {
				delta = -1
			}
			a.Store.DB.Exec("INSERT INTO analytics(day,channel_id,subscribers)VALUES(?,?,?) ON CONFLICT(day,channel_id)DO UPDATE SET subscribers=subscribers+excluded.subscribers", time.Now().UTC().Format("2006-01-02"), ch, delta)
		}
	}
	a.failure(w, e)
}
func (a *App) Progress(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	v, e := a.video(r.PathValue("id"), uid)
	if e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	var in struct {
		Position float64 `json:"position"`
		Seconds  float64 `json:"seconds"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Position < 0 || in.Position > v.Duration+1 || in.Seconds < 0 || in.Seconds > 30 {
		problem(w, 400, "Invalid playback progress")
		return
	}
	var updated int64
	var counted bool
	a.Store.DB.QueryRow("SELECT updated,counted FROM history WHERE user_id=? AND video_id=?", uid, v.ID).Scan(&updated, &counted)
	elapsed := float64(now()-updated) / 1000
	if updated == 0 {
		elapsed = 0
	}
	seconds := in.Seconds
	if seconds > elapsed {
		seconds = elapsed
	}
	if seconds < 0 {
		seconds = 0
	}
	delta := 0
	if !counted && in.Position >= min(10, v.Duration*.5) {
		delta = 1
		counted = true
	}
	_, e = a.Store.DB.Exec("INSERT INTO history(user_id,video_id,position,seconds,updated,counted)VALUES(?,?,?,?,?,?) ON CONFLICT(user_id,video_id)DO UPDATE SET position=excluded.position,seconds=history.seconds+excluded.seconds,updated=excluded.updated,counted=excluded.counted", uid, v.ID, in.Position, seconds, now(), counted)
	if e == nil {
		_, e = a.Store.DB.Exec("UPDATE videos SET views=views+?,watch_seconds=watch_seconds+? WHERE id=?", delta, seconds, v.ID)
	}
	a.failure(w, e)
}
func (a *App) ClearHistory(w http.ResponseWriter, r *http.Request) {
	_, e := a.Store.DB.Exec("DELETE FROM history WHERE user_id=?", who(r).User.ID)
	a.failure(w, e)
}

type Comment struct {
	ID      string `json:"id"`
	VideoID string `json:"video_id"`
	UserID  string `json:"user_id"`
	Parent  string `json:"parent"`
	Body    string `json:"body"`
	Name    string `json:"name"`
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
	Hidden  bool   `json:"hidden"`
}

func (a *App) Comments(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	v, e := a.video(r.PathValue("id"), uid)
	if e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	rows, e := a.Store.DB.Query("SELECT c.id,c.video_id,c.user_id,c.parent,c.body,u.name,c.created,c.updated,c.hidden FROM comments c JOIN users u ON u.id=c.user_id WHERE video_id=? AND (hidden=0 OR c.user_id=? OR ?=?) ORDER BY created LIMIT 500", v.ID, uid, uid, v.Owner)
	if e != nil {
		problem(w, 500, "Comments unavailable")
		return
	}
	defer rows.Close()
	out := []Comment{}
	for rows.Next() {
		var c Comment
		if rows.Scan(&c.ID, &c.VideoID, &c.UserID, &c.Parent, &c.Body, &c.Name, &c.Created, &c.Updated, &c.Hidden) == nil {
			out = append(out, c)
		}
	}
	respond(w, 200, out)
}
func (a *App) AddComment(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	v, e := a.video(r.PathValue("id"), uid)
	if e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	var in struct {
		Body   string `json:"body"`
		Parent string `json:"parent"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if len(in.Body) == 0 || len(in.Body) > 5000 {
		problem(w, 400, "Comment must be 1–5000 bytes")
		return
	}
	var n int
	a.Store.DB.QueryRow("SELECT count(*) FROM comments WHERE user_id=? AND created>?", uid, now()-60000).Scan(&n)
	if n >= 5 {
		problem(w, 429, "Please wait before commenting again")
		return
	}
	if in.Parent != "" {
		var p string
		if a.Store.DB.QueryRow("SELECT id FROM comments WHERE id=? AND video_id=? AND parent='' AND hidden=0", in.Parent, v.ID).Scan(&p) != nil {
			problem(w, 400, "Reply to a visible top-level comment")
			return
		}
	}
	id := ID()
	_, e = a.Store.DB.Exec("INSERT INTO comments(id,video_id,user_id,parent,body,created,updated)VALUES(?,?,?,?,?,?,?)", id, v.ID, uid, in.Parent, in.Body, now(), now())
	if e != nil {
		a.failure(w, e)
		return
	}
	respond(w, 201, map[string]string{"id": id})
}
func (a *App) EditComment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Body string `json:"body"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Body)) == 0 || len(in.Body) > 5000 {
		problem(w, 400, "Invalid comment")
		return
	}
	uid := who(r).User.ID
	var vid string
	if a.Store.DB.QueryRow("SELECT video_id FROM comments WHERE id=? AND user_id=?", r.PathValue("id"), uid).Scan(&vid) != nil {
		problem(w, 404, "Comment unavailable")
		return
	}
	if _, e := a.video(vid, uid); e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	_, e := a.Store.DB.Exec("UPDATE comments SET body=?,updated=? WHERE id=? AND user_id=?", in.Body, now(), r.PathValue("id"), uid)
	a.failure(w, e)
}
func (a *App) RemoveComment(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	var vid, author string
	if a.Store.DB.QueryRow("SELECT video_id,user_id FROM comments WHERE id=?", r.PathValue("id")).Scan(&vid, &author) != nil {
		problem(w, 404, "Comment unavailable")
		return
	}
	v, e := a.video(vid, uid)
	if e != nil || uid != author && uid != v.Owner {
		problem(w, 404, "Comment unavailable")
		return
	}
	_, e = a.Store.DB.Exec("UPDATE comments SET hidden=1,body='[removed]' WHERE id=? OR parent=?", r.PathValue("id"), r.PathValue("id"))
	a.Store.Audit(uid, "comment.remove", r.PathValue("id"))
	a.failure(w, e)
}

type Playlist struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	Count       int    `json:"count"`
}

func (a *App) canPlaylist(id, uid string) bool {
	var owner, vis string
	if a.Store.DB.QueryRow("SELECT owner,visibility FROM playlists WHERE id=?", id).Scan(&owner, &vis) != nil {
		return false
	}
	return owner == uid && uid != "" || vis != "private" && (uid != "" || a.Config.PublicAccess)
}
func (a *App) Playlists(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	where := "p.owner=?"
	args := []any{uid}
	if owner := r.URL.Query().Get("owner"); owner != "" && owner != uid {
		where = "p.owner=? AND p.visibility='public' AND (?<>'' OR ?)"
		args = []any{owner, uid, a.Config.PublicAccess}
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		where += " AND p.rowid IN(SELECT rowid FROM playlist_search WHERE playlist_search MATCH ?)"
		args = append(args, `"`+strings.ReplaceAll(q, `"`, `""`)+`"`)
	}
	rows, e := a.Store.DB.Query("SELECT p.id,p.owner,p.title,p.description,p.visibility,(SELECT count(*) FROM playlist_videos WHERE playlist_id=p.id) FROM playlists p WHERE "+where+" ORDER BY created DESC", args...)
	if e != nil {
		problem(w, 500, "Playlists unavailable")
		return
	}
	defer rows.Close()
	out := []Playlist{}
	for rows.Next() {
		var p Playlist
		if rows.Scan(&p.ID, &p.Owner, &p.Title, &p.Description, &p.Visibility, &p.Count) == nil {
			out = append(out, p)
		}
	}
	respond(w, 200, out)
}
func (a *App) Playlist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !a.canPlaylist(id, who(r).User.ID) {
		problem(w, 404, "Playlist unavailable")
		return
	}
	var p Playlist
	a.Store.DB.QueryRow("SELECT id,owner,title,description,visibility FROM playlists WHERE id=?", id).Scan(&p.ID, &p.Owner, &p.Title, &p.Description, &p.Visibility)
	respond(w, 200, p)
}
func (a *App) SavePlaylist(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	var in struct{ Title, Description, Visibility string }
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Title)) == 0 || len(in.Title) > 200 || len(in.Description) > 5000 || !visibility(in.Visibility) || in.Visibility == "public" && !a.Config.PublicUploads {
		problem(w, 400, "Invalid playlist")
		return
	}
	id := r.PathValue("id")
	if id != "" {
		var owner string
		if a.Store.DB.QueryRow("SELECT owner FROM playlists WHERE id=?", id).Scan(&owner) != nil || owner != uid {
			problem(w, 404, "Playlist unavailable")
			return
		}
	} else {
		id = ID()
	}
	_, e := a.Store.DB.Exec("INSERT INTO playlists(id,owner,title,description,visibility,created)VALUES(?,?,?,?,?,?) ON CONFLICT(id)DO UPDATE SET title=excluded.title,description=excluded.description,visibility=excluded.visibility", id, uid, in.Title, in.Description, in.Visibility, now())
	if e != nil {
		a.failure(w, e)
		return
	}
	a.Store.DB.Exec("DELETE FROM playlist_search WHERE rowid=(SELECT rowid FROM playlists WHERE id=?);INSERT INTO playlist_search(rowid,title,description)SELECT rowid,title,description FROM playlists WHERE id=?", id, id)
	respond(w, 200, map[string]string{"id": id})
}
func (a *App) playlistOwner(w http.ResponseWriter, r *http.Request) bool {
	var owner string
	if a.Store.DB.QueryRow("SELECT owner FROM playlists WHERE id=?", r.PathValue("id")).Scan(&owner) != nil || owner != who(r).User.ID {
		problem(w, 404, "Playlist unavailable")
		return false
	}
	return true
}
func (a *App) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	if !a.playlistOwner(w, r) {
		return
	}
	a.Store.DB.Exec("DELETE FROM playlist_search WHERE rowid=(SELECT rowid FROM playlists WHERE id=?)", r.PathValue("id"))
	_, e := a.Store.DB.Exec("DELETE FROM playlists WHERE id=?", r.PathValue("id"))
	a.failure(w, e)
}
func (a *App) PlaylistAdd(w http.ResponseWriter, r *http.Request) {
	if !a.playlistOwner(w, r) {
		return
	}
	var in struct {
		VideoID string `json:"video_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if _, e := a.video(in.VideoID, who(r).User.ID); e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	_, e := a.Store.DB.Exec("INSERT OR IGNORE INTO playlist_videos VALUES(?,?,(SELECT count(*) FROM playlist_videos WHERE playlist_id=?))", r.PathValue("id"), in.VideoID, r.PathValue("id"))
	a.failure(w, e)
}
func (a *App) PlaylistRemove(w http.ResponseWriter, r *http.Request) {
	if !a.playlistOwner(w, r) {
		return
	}
	_, e := a.Store.DB.Exec("DELETE FROM playlist_videos WHERE playlist_id=? AND video_id=?", r.PathValue("id"), r.PathValue("video"))
	a.failure(w, e)
}
func (a *App) Notifications(w http.ResponseWriter, r *http.Request) {
	rows, e := a.Store.DB.Query("SELECT id,video_id,body,created,seen FROM notifications WHERE user_id=? ORDER BY created DESC LIMIT 100", who(r).User.ID)
	if e != nil {
		problem(w, 500, "Notifications unavailable")
		return
	}
	type n struct {
		ID      string `json:"id"`
		Video   string `json:"video_id"`
		Body    string `json:"body"`
		Created int64  `json:"created"`
		Seen    bool   `json:"seen"`
	}
	all := []n{}
	for rows.Next() {
		var v n
		if rows.Scan(&v.ID, &v.Video, &v.Body, &v.Created, &v.Seen) == nil {
			all = append(all, v)
		}
	}
	rows.Close()
	out := []n{}
	for _, v := range all {
		if _, e := a.video(v.Video, who(r).User.ID); e == nil {
			out = append(out, v)
		}
	}
	respond(w, 200, out)
}
func (a *App) ReadNotifications(w http.ResponseWriter, r *http.Request) {
	_, e := a.Store.DB.Exec("UPDATE notifications SET seen=1 WHERE user_id=?", who(r).User.ID)
	a.failure(w, e)
}
func (a *App) Preferences(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Personalize bool `json:"personalize"`
	}
	if !decode(w, r, &in) {
		return
	}
	_, e := a.Store.DB.Exec("UPDATE users SET personalize=? WHERE id=?", in.Personalize, who(r).User.ID)
	a.failure(w, e)
}
func (a *App) Studio(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	var views int64
	var seconds float64
	var videos, likes, subscribers, unique int
	a.Store.DB.QueryRow("SELECT count(*),COALESCE(SUM(views),0),COALESCE(SUM(watch_seconds),0) FROM videos WHERE owner=? AND deleted=0", uid).Scan(&videos, &views, &seconds)
	a.Store.DB.QueryRow("SELECT count(*) FROM likes l JOIN videos v ON v.id=l.video_id WHERE v.owner=? AND v.deleted=0", uid).Scan(&likes)
	a.Store.DB.QueryRow("SELECT count(*) FROM subscriptions s JOIN channels c ON c.id=s.channel_id WHERE c.owner=?", uid).Scan(&subscribers)
	var distinct any = nil
	if a.Config.DetailedAnalytics {
		a.Store.DB.QueryRow("SELECT count(DISTINCT h.user_id) FROM history h JOIN videos v ON v.id=h.video_id WHERE v.owner=? AND h.counted=1 AND v.deleted=0", uid).Scan(&unique)
		distinct = unique
	}
	type point struct {
		Day         string `json:"day"`
		Subscribers int    `json:"subscribers"`
	}
	growth := []point{}
	rows, _ := a.Store.DB.Query("SELECT a.day,a.subscribers FROM analytics a JOIN channels c ON c.id=a.channel_id WHERE c.owner=? ORDER BY day DESC LIMIT 90", uid)
	if rows != nil {
		for rows.Next() {
			var p point
			if rows.Scan(&p.Day, &p.Subscribers) == nil {
				growth = append(growth, p)
			}
		}
		rows.Close()
	}
	respond(w, 200, map[string]any{"videos": videos, "views": views, "watch_seconds": seconds, "likes": likes, "subscribers": subscribers, "unique_authenticated_viewers_in_retention_window": distinct, "subscriber_growth": growth, "measurement": "Views count once per signed-in account while retained in history. Watch time is client-reported and capped by elapsed server time; anonymous views are not counted."})
}
func (a *App) Report(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	v, e := a.video(r.PathValue("id"), uid)
	if e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Reason)) < 3 || len(in.Reason) > 2000 {
		problem(w, 400, "Give a reason up to 2000 bytes")
		return
	}
	var n int
	a.Store.DB.QueryRow("SELECT count(*) FROM reports WHERE user_id=? AND created>?", uid, now()-3600000).Scan(&n)
	if n >= 10 {
		problem(w, 429, "Report limit reached")
		return
	}
	_, e = a.Store.DB.Exec("INSERT INTO reports(id,user_id,video_id,reason,created)VALUES(?,?,?,?,?)", ID(), uid, v.ID, in.Reason, now())
	a.failure(w, e)
}
