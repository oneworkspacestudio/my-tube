package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const chunkSize int64 = 8 << 20

type Upload struct {
	ID         string `json:"id"`
	Owner      string `json:"-"`
	Channel    string `json:"channel_id"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	Visibility string `json:"visibility"`
	Size       int64  `json:"size"`
	Offset     int64  `json:"offset"`
	Created    int64  `json:"created"`
	VideoID    string `json:"video_id"`
}

func (a *App) upload(id, uid string) (Upload, error) {
	var u Upload
	e := a.Store.DB.QueryRow("SELECT id,owner,channel_id,name,title,visibility,size,offset,created,video_id FROM uploads WHERE id=? AND owner=?", id, uid).Scan(&u.ID, &u.Owner, &u.Channel, &u.Name, &u.Title, &u.Visibility, &u.Size, &u.Offset, &u.Created, &u.VideoID)
	return u, e
}
func (a *App) incoming(id string) string {
	return filepath.Join(a.Config.MediaDir, "incoming", id+".part")
}
func (a *App) reserve(uid string, size int64) error {
	var used, quota, total int64
	e := a.Store.DB.QueryRow(`SELECT quota_mb,(SELECT COALESCE(SUM(size),0) FROM videos WHERE owner=?)+(SELECT COALESCE(SUM(size),0) FROM uploads WHERE owner=? AND video_id='') FROM users WHERE id=?`, uid, uid, uid).Scan(&quota, &used)
	if e != nil {
		return e
	}
	if used+size > quota<<20 {
		return fmt.Errorf("Account quota exceeded")
	}
	a.Store.DB.QueryRow("SELECT (SELECT COALESCE(SUM(size),0) FROM videos)+(SELECT COALESCE(SUM(size),0) FROM uploads WHERE video_id='')").Scan(&total)
	if total+size > a.Config.StorageMB<<20 {
		return fmt.Errorf("Server quota exceeded")
	}
	actual, e := treeSize(a.Config.MediaDir)
	if e != nil {
		return e
	}
	if actual+size > a.Config.StorageMB<<20 {
		return fmt.Errorf("Media storage limit exceeded")
	}
	return nil
}
func treeSize(dir string) (int64, error) {
	var n int64
	e := filepath.WalkDir(dir, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("Symlinks not allowed in managed media")
		}
		if !d.IsDir() {
			i, e := d.Info()
			if e != nil {
				return e
			}
			n += i.Size()
		}
		return nil
	})
	return n, e
}
func (a *App) BeginUpload(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string `json:"name"`
		Title      string `json:"title"`
		Size       int64  `json:"size"`
		Visibility string `json:"visibility"`
	}
	if !decode(w, r, &in) {
		return
	}
	u, e := a.begin(who(r).User.ID, in.Name, in.Title, in.Size, in.Visibility)
	if e != nil {
		problem(w, 400, e.Error())
		return
	}
	respond(w, 201, u)
}
func (a *App) begin(uid, name, title string, size int64, vis string) (Upload, error) {
	var u Upload
	ext := strings.ToLower(filepath.Ext(name))
	allowed := false
	for _, v := range a.Config.AllowedFormats {
		if v == ext {
			allowed = true
		}
	}
	if !allowed {
		return u, fmt.Errorf("This container is disabled by the administrator")
	}
	if ext != ".mp4" && ext != ".mov" && ext != ".mkv" && ext != ".webm" {
		return u, fmt.Errorf("Supported containers: MP4, MOV, MKV, WebM")
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, "\\/:\x00") || len(name) > 240 || size <= 0 || size > a.Config.UploadMB<<20 || !visibility(vis) || vis == "public" && !a.Config.PublicUploads {
		return u, fmt.Errorf("Invalid filename, size or visibility")
	}
	ch, e := a.ownerChannel(uid)
	if e != nil {
		return u, fmt.Errorf("Create a channel first")
	}
	if strings.TrimSpace(title) == "" {
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	if len(title) > 200 {
		return u, fmt.Errorf("Title too long")
	}
	var active int
	a.Store.DB.QueryRow("SELECT count(*) FROM uploads WHERE owner=? AND video_id=''", uid).Scan(&active)
	if active >= 20 {
		return u, fmt.Errorf("Finish or cancel existing uploads first")
	}
	if e = a.reserve(uid, size); e != nil {
		return u, e
	}
	u = Upload{ID: ID(), Owner: uid, Channel: ch, Name: name, Title: title, Visibility: vis, Size: size, Created: now()}
	if e = os.MkdirAll(filepath.Dir(a.incoming(u.ID)), 0700); e != nil {
		return u, e
	}
	f, e := os.OpenFile(a.incoming(u.ID), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return u, e
	}
	e = f.Close()
	if e != nil {
		return u, e
	}
	_, e = a.Store.DB.Exec("INSERT INTO uploads(id,owner,channel_id,name,title,visibility,size,created)VALUES(?,?,?,?,?,?,?,?)", u.ID, uid, ch, name, title, vis, size, u.Created)
	if e != nil {
		os.Remove(a.incoming(u.ID))
	}
	return u, e
}
func (a *App) Uploads(w http.ResponseWriter, r *http.Request) {
	rows, e := a.Store.DB.Query("SELECT id FROM uploads WHERE owner=? AND video_id='' ORDER BY created DESC", who(r).User.ID)
	if e != nil {
		problem(w, 500, "Uploads unavailable")
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	out := []Upload{}
	for _, id := range ids {
		u, e := a.upload(id, who(r).User.ID)
		if e == nil {
			out = append(out, u)
		}
	}
	respond(w, 200, out)
}
func (a *App) UploadStatus(w http.ResponseWriter, r *http.Request) {
	u, e := a.upload(r.PathValue("id"), who(r).User.ID)
	if e != nil {
		problem(w, 404, "Upload unavailable")
		return
	}
	respond(w, 200, u)
}
func (a *App) UploadChunk(w http.ResponseWriter, r *http.Request) {
	u, e := a.upload(r.PathValue("id"), who(r).User.ID)
	if e != nil || u.VideoID != "" {
		problem(w, 404, "Upload unavailable")
		return
	}
	offset, e := strconv.ParseInt(r.Header.Get("Upload-Offset"), 10, 64)
	if e != nil || offset != u.Offset {
		w.Header().Set("Upload-Offset", strconv.FormatInt(u.Offset, 10))
		problem(w, 409, "Offset mismatch; fetch upload status before retrying")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, chunkSize)
	b, e := io.ReadAll(r.Body)
	if e != nil || len(b) == 0 || int64(len(b)) > u.Size-u.Offset {
		problem(w, 400, "Invalid chunk size")
		return
	}
	f, e := os.OpenFile(a.incoming(u.ID), os.O_WRONLY, 0600)
	if e != nil {
		problem(w, 500, "Upload staging file unavailable")
		return
	}
	_, e = f.WriteAt(b, u.Offset)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil || ce != nil {
		problem(w, 507, "Cannot persist upload chunk")
		return
	}
	newOffset := u.Offset + int64(len(b))
	_, e = a.Store.DB.Exec("UPDATE uploads SET offset=? WHERE id=?", newOffset, u.ID)
	if e != nil {
		os.Truncate(a.incoming(u.ID), u.Offset)
		problem(w, 500, "Chunk checkpoint failed")
		return
	}
	u.Offset = newOffset
	respond(w, 200, u)
}
func (a *App) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	u, e := a.upload(r.PathValue("id"), who(r).User.ID)
	if e != nil {
		problem(w, 404, "Upload unavailable")
		return
	}
	if u.VideoID != "" {
		respond(w, 200, map[string]string{"id": u.VideoID})
		return
	}
	id, e := a.finish(r, u)
	if e != nil {
		problem(w, 400, e.Error())
		return
	}
	respond(w, 201, map[string]string{"id": id})
}
func (a *App) finish(r *http.Request, u Upload) (string, error) {
	if u.Offset != u.Size {
		return "", fmt.Errorf("Upload is incomplete")
	}
	f, e := os.Open(a.incoming(u.ID))
	if e != nil {
		return "", e
	}
	h := sha256.New()
	_, e = io.Copy(h, f)
	f.Close()
	if e != nil {
		return "", e
	}
	hash := hex.EncodeToString(h.Sum(nil))
	var existing string
	if a.Store.DB.QueryRow("SELECT id FROM videos WHERE owner=? AND sha256=?", u.Owner, hash).Scan(&existing) == nil {
		return "", fmt.Errorf("Duplicate video in your library: %s", existing)
	}
	meta, e := a.inspect(r.Context(), a.incoming(u.ID))
	if e != nil {
		return "", e
	}
	if a.Config.ScanProgram != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		e = a.run(ctx, a.Config.ScanProgram, []string{a.incoming(u.ID)}, nil)
		cancel()
		if e != nil {
			return "", fmt.Errorf("Malware scanner rejected the upload or failed")
		}
	}
	id := u.ID
	original := id + strings.ToLower(filepath.Ext(u.Name))
	dest := filepath.Join(a.Config.MediaDir, "originals", original)
	if e = os.Rename(a.incoming(u.ID), dest); e != nil {
		return "", e
	}
	tx, e := a.Store.DB.Begin()
	if e != nil {
		os.Rename(dest, a.incoming(u.ID))
		return "", e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`INSERT INTO videos(id,owner,channel_id,title,visibility,original,filename,size,sha256,duration,width,height,direct,created)VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, u.Owner, u.Channel, u.Title, u.Visibility, original, u.Name, u.Size, hash, meta.Duration, meta.Width, meta.Height, meta.Direct, now())
	if e == nil {
		_, e = tx.Exec("INSERT INTO jobs(id,video_id,status,updated)VALUES(?,?,'queued',?)", ID(), id, now())
	}
	if e == nil {
		_, e = tx.Exec("UPDATE uploads SET video_id=? WHERE id=?", id, u.ID)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		tx.Rollback()
		os.Rename(dest, a.incoming(u.ID))
		return "", e
	}
	a.indexVideo(id)
	if u.Visibility == "public" {
		a.Store.DB.Exec(`INSERT INTO notifications SELECT lower(hex(randomblob(16))),user_id,?,?,?,0 FROM subscriptions WHERE channel_id=? AND user_id<>?`, id, u.Title, now(), u.Channel, u.Owner)
	}
	a.Store.Audit(u.Owner, "video.upload", id)
	return id, nil
}
func (a *App) CancelUpload(w http.ResponseWriter, r *http.Request) {
	u, e := a.upload(r.PathValue("id"), who(r).User.ID)
	if e != nil || u.VideoID != "" {
		problem(w, 404, "Upload unavailable")
		return
	}
	if e = os.Remove(a.incoming(u.ID)); e != nil && !os.IsNotExist(e) {
		problem(w, 500, "Cannot remove staging file")
		return
	}
	_, e = a.Store.DB.Exec("DELETE FROM uploads WHERE id=?", u.ID)
	a.failure(w, e)
}
