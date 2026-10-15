package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// StorageAdapter keeps user-scoped external APIs independent of video metadata.
// Imports are explicit owned snapshots; they never point around Quantum ACLs.
type StorageAdapter interface {
	List(context.Context, string, string) (json.RawMessage, error)
	Read(context.Context, string, string) (io.ReadCloser, int64, error)
	Write(context.Context, string, string, io.Reader, int64) error
}
type Quantum struct {
	Base, Token string
	Client      *http.Client
}

func (q Quantum) request(ctx context.Context, method, endpoint string, params url.Values, body io.Reader, size int64) (*http.Response, error) {
	req, e := http.NewRequestWithContext(ctx, method, strings.TrimRight(q.Base, "/")+endpoint+"?"+params.Encode(), body)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+q.Token)
	req.Header.Set("Content-Type", "application/octet-stream")
	if size >= 0 {
		req.ContentLength = size
	}
	resp, e := q.Client.Do(req)
	if e != nil {
		return nil, errors.New("Drive connection failed")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("Drive denied the operation (HTTP %d)", resp.StatusCode)
	}
	return resp, nil
}
func (q Quantum) List(ctx context.Context, source, p string) (json.RawMessage, error) {
	r, e := q.request(ctx, "GET", "/api/resources", url.Values{"source": {source}, "path": {p}}, nil, 0)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	b, e := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if e != nil || !json.Valid(b) {
		return nil, errors.New("Invalid Drive listing")
	}
	return b, nil
}
func (q Quantum) Read(ctx context.Context, source, p string) (io.ReadCloser, int64, error) {
	r, e := q.request(ctx, "GET", "/api/resources/download", url.Values{"source": {source}, "file": {p}}, nil, 0)
	if e != nil {
		return nil, 0, e
	}
	return r.Body, r.ContentLength, nil
}
func (q Quantum) Write(ctx context.Context, source, p string, body io.Reader, size int64) error {
	r, e := q.request(ctx, "POST", "/api/resources", url.Values{"source": {source}, "path": {p}, "override": {"false"}, "isDir": {"false"}}, body, size)
	if e != nil {
		return e
	}
	r.Body.Close()
	return nil
}
func (a *App) drive(r *http.Request) (StorageAdapter, error) {
	if a.Config.DriveURL == "" {
		return nil, errors.New("Administrator must configure drive_url")
	}
	token := r.Header.Get("X-Drive-Token")
	if token == "" {
		if c, e := r.Cookie("filebrowser_quantum_jwt"); e == nil {
			token = c.Value
		}
	}
	if token == "" || len(token) > 16384 || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("Sign in to My Drive on this Workspace origin, or provide your own scoped Drive API token")
	}
	q := Quantum{Base: a.Config.DriveURL, Token: token, Client: &http.Client{Timeout: 30 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	res, e := q.request(ctx, "GET", "/api/users", url.Values{"username": {"self"}}, nil, 0)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	var self struct {
		Username    string `json:"username"`
		Permissions struct {
			Admin bool `json:"admin"`
		} `json:"permissions"`
		Perm struct {
			Admin bool `json:"admin"`
		} `json:"perm"`
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&self); e != nil || self.Username == "" {
		return nil, errors.New("Drive could not verify the token owner")
	}
	if self.Permissions.Admin || self.Perm.Admin {
		return nil, errors.New("Use a scoped non-administrator Drive token for media import/export")
	}
	return q, nil
}

type driveInput struct {
	Source     string `json:"source"`
	Path       string `json:"path"`
	Title      string `json:"title"`
	Visibility string `json:"visibility"`
	Size       int64  `json:"size"`
}

func drivePath(in driveInput) bool {
	if in.Source == "" || len(in.Source) > 200 || len(in.Path) > 4096 || strings.ContainsAny(in.Path, "\\\x00") || strings.Contains(in.Path, ":") {
		return false
	}
	for _, s := range strings.Split(in.Path, "/") {
		if s == ".." {
			return false
		}
	}
	return strings.HasPrefix(in.Path, "/")
}
func (a *App) DriveList(w http.ResponseWriter, r *http.Request) {
	var in driveInput
	if !decode(w, r, &in) {
		return
	}
	if !drivePath(in) {
		problem(w, 400, "Choose a Drive source and absolute scoped path")
		return
	}
	s, e := a.drive(r)
	if e != nil {
		problem(w, 403, e.Error())
		return
	}
	b, e := s.List(r.Context(), in.Source, in.Path)
	if e != nil {
		problem(w, 403, e.Error())
		return
	}
	respond(w, 200, b)
}
func (a *App) DriveImport(w http.ResponseWriter, r *http.Request) {
	var in driveInput
	if !decode(w, r, &in) {
		return
	}
	if !drivePath(in) {
		problem(w, 400, "Invalid Drive path")
		return
	}
	s, e := a.drive(r)
	if e != nil {
		problem(w, 403, e.Error())
		return
	}
	body, size, e := s.Read(r.Context(), in.Source, in.Path)
	if e != nil {
		problem(w, 403, e.Error())
		return
	}
	defer body.Close()
	if size < 1 {
		size = in.Size
	}
	if in.Visibility == "" {
		in.Visibility = "private"
	}
	u, e := a.begin(who(r).User.ID, path.Base(in.Path), in.Title, size, in.Visibility)
	if e != nil {
		problem(w, 400, e.Error())
		return
	}
	f, e := os.OpenFile(a.incoming(u.ID), os.O_WRONLY, 0600)
	if e != nil {
		problem(w, 500, "Staging unavailable")
		return
	}
	n, e := io.Copy(f, io.LimitReader(body, size+1))
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil || ce != nil || n != size {
		os.Remove(a.incoming(u.ID))
		a.Store.DB.Exec("DELETE FROM uploads WHERE id=?", u.ID)
		problem(w, 400, "Drive import interrupted or size mismatch; original unchanged")
		return
	}
	u.Offset = n
	a.Store.DB.Exec("UPDATE uploads SET offset=? WHERE id=?", n, u.ID)
	id, e := a.finish(r, u)
	if e != nil {
		problem(w, 400, e.Error())
		return
	}
	respond(w, 201, map[string]string{"id": id})
}
func (a *App) DriveExport(w http.ResponseWriter, r *http.Request) {
	v, e := a.video(r.PathValue("id"), who(r).User.ID)
	if e != nil || v.Owner != who(r).User.ID {
		problem(w, 404, "Owned video unavailable")
		return
	}
	var in driveInput
	if !decode(w, r, &in) {
		return
	}
	if !drivePath(in) {
		problem(w, 400, "Invalid Drive path")
		return
	}
	s, e := a.drive(r)
	if e != nil {
		problem(w, 403, e.Error())
		return
	}
	f, e := os.Open(a.original(v))
	if e != nil {
		problem(w, 404, "Original missing")
		return
	}
	defer f.Close()
	if e = s.Write(r.Context(), in.Source, in.Path, f, v.Size); e != nil {
		problem(w, 403, e.Error())
		return
	}
	a.Store.Audit(who(r).User.ID, "drive.export", v.ID)
	respond(w, 200, map[string]bool{"ok": true})
}
