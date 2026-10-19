package app

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type App struct {
	Config                   Config
	Store                    *Store
	Assets                   fs.FS
	Logger                   *log.Logger
	oidc                     authProvider
	localToken               string
	localMu, writeMu, rateMu sync.Mutex
	rates                    map[string]rate
	cancel                   context.CancelFunc
	workers                  sync.WaitGroup
	helpers                  mediaHelpers
}
type rate struct {
	count int
	since time.Time
}

func New(c Config, assets fs.FS, logger *log.Logger) (*App, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	s, e := OpenStore(c.DataDir)
	if e != nil {
		return nil, e
	}
	if logger == nil {
		logger = log.New(os.Stderr, "mytube ", log.LstdFlags)
	}
	for _, p := range []string{"originals", "streaming", "thumbnails", "subtitles"} {
		if e = os.MkdirAll(filepath.Join(c.MediaDir, p), 0700); e != nil {
			s.Close()
			return nil, e
		}
	}
	a := &App{Config: c, Store: s, Assets: assets, Logger: logger, rates: map[string]rate{}}
	if e = a.Recover(); e != nil {
		s.Close()
		return nil, e
	}
	return a, nil
}
func (a *App) StartWorkers() {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	if a.Config.Transcode {
		a.queueUnoptimised()
		if a.Config.AutoHelpers {
			a.workers.Add(1)
			go func() {
				defer a.workers.Done()
				if e := a.prepareMedia(ctx); e != nil {
					a.Logger.Print("Adaptive video tools are not ready; originals remain available")
				}
			}()
		}
	}
	for i := 0; i < a.Config.Workers; i++ {
		a.workers.Add(1)
		go func() { defer a.workers.Done(); a.worker(ctx) }()
	}
}
func (a *App) Close() error {
	if a.cancel != nil {
		a.cancel()
		a.workers.Wait()
	}
	return a.Store.Close()
}
func (a *App) LocalURL() string {
	a.localMu.Lock()
	defer a.localMu.Unlock()
	t := secret()
	a.localToken = digest(t)
	return a.Config.PublicURL + "/auth/local?token=" + t
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		problem(w, 400, "Invalid JSON")
		return false
	}
	if d.Decode(new(any)) != io.EOF {
		problem(w, 400, "Expected one JSON value")
		return false
	}
	return true
}
func (a *App) Optional(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, e := r.Cookie("mytube_session"); e == nil {
			a.Require(next)(w, r)
		} else {
			next(w, r)
		}
	}
}
func (a *App) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if e := a.Store.DB.PingContext(r.Context()); e != nil {
			problem(w, 503, "Database unavailable")
			return
		}
		respond(w, 200, map[string]any{"app": "my-tube", "version": Version, "status": "ok"})
	})
	m.HandleFunc("GET /auth/login", a.Login)
	m.HandleFunc("GET /auth/callback", a.Callback)
	m.HandleFunc("GET /auth/local", a.LocalLogin)
	routes := []struct {
		pattern  string
		fn       http.HandlerFunc
		optional bool
	}{
		{"GET /api/me", a.Me, true}, {"POST /api/logout", a.Logout, false},
		{"GET /api/videos", a.ListVideos, true}, {"GET /api/videos/{id}", a.GetVideo, true}, {"PUT /api/videos/{id}", a.EditVideo, false}, {"DELETE /api/videos/{id}", a.DeleteVideo, false},
		{"GET /api/videos/{id}/media/{file}", a.Media, true}, {"POST /api/videos/{id}/thumbnail", a.Thumbnail, false}, {"POST /api/videos/{id}/subtitles", a.Subtitles, false}, {"PUT /api/videos/{id}/access", a.VideoAccess, false}, {"POST /api/videos/{id}/retry", a.RetryJob, false}, {"POST /api/videos/{id}/replace", a.ReplaceVideo, false},
		{"POST /api/uploads", a.BeginUpload, false}, {"GET /api/uploads", a.Uploads, false}, {"GET /api/uploads/{id}", a.UploadStatus, false}, {"PATCH /api/uploads/{id}", a.UploadChunk, false}, {"POST /api/uploads/{id}/complete", a.CompleteUpload, false}, {"DELETE /api/uploads/{id}", a.CancelUpload, false},
		{"GET /api/channels", a.Channels, true}, {"GET /api/channels/{id}", a.Channel, true}, {"POST /api/channel", a.SaveChannel, false}, {"POST /api/channel/image/{kind}", a.ChannelImage, false}, {"GET /api/channels/{id}/image/{kind}", a.GetChannelImage, true},
		{"POST /api/videos/{id}/like", a.Like, false}, {"POST /api/videos/{id}/later", a.Later, false}, {"POST /api/videos/{id}/progress", a.Progress, false}, {"DELETE /api/history", a.ClearHistory, false}, {"POST /api/channels/{id}/subscribe", a.Subscribe, false},
		{"GET /api/videos/{id}/comments", a.Comments, true}, {"POST /api/videos/{id}/comments", a.AddComment, false}, {"PUT /api/comments/{id}", a.EditComment, false}, {"DELETE /api/comments/{id}", a.RemoveComment, false},
		{"GET /api/playlists", a.Playlists, true}, {"POST /api/playlists", a.SavePlaylist, false}, {"GET /api/playlists/{id}", a.Playlist, true}, {"PUT /api/playlists/{id}", a.SavePlaylist, false}, {"DELETE /api/playlists/{id}", a.DeletePlaylist, false}, {"POST /api/playlists/{id}/videos", a.PlaylistAdd, false}, {"DELETE /api/playlists/{id}/videos/{video}", a.PlaylistRemove, false},
		{"GET /api/notifications", a.Notifications, false}, {"POST /api/notifications/read", a.ReadNotifications, false}, {"PUT /api/preferences", a.Preferences, false}, {"GET /api/studio", a.Studio, false}, {"POST /api/videos/{id}/report", a.Report, false},
		{"GET /api/admin", a.Admin, false}, {"GET /api/admin/update", a.CheckUpdate, false}, {"PUT /api/admin/config", a.AdminConfig, false}, {"PUT /api/admin/users/{id}", a.AdminUser, false}, {"POST /api/admin/reconcile", a.Reconcile, false}, {"PUT /api/admin/reports/{id}", a.ResolveReport, false},
		{"POST /api/drive/list", a.DriveList, false}, {"POST /api/drive/import", a.DriveImport, false}, {"POST /api/videos/{id}/drive-export", a.DriveExport, false},
	}
	for _, rt := range routes {
		fn := a.Require(rt.fn)
		if rt.optional {
			fn = a.Optional(rt.fn)
		}
		m.HandleFunc(rt.pattern, fn)
	}
	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { problem(w, 404, "Unknown API endpoint") })
	m.HandleFunc("/auth/", func(w http.ResponseWriter, r *http.Request) { problem(w, 404, "Unknown auth endpoint") })
	m.HandleFunc("/", a.Static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; media-src 'self' blob:; worker-src 'self' blob:; connect-src 'self'; frame-ancestors 'self'; object-src 'none'; base-uri 'self'; form-action 'self'")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(a.Config.PublicURL, "https://") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		a.rateMu.Lock()
		rt := a.rates[host]
		if time.Since(rt.since) > time.Minute {
			rt = rate{since: time.Now()}
		}
		rt.count++
		a.rates[host] = rt
		limited := rt.count > 2400
		if len(a.rates) > 10000 {
			for k, v := range a.rates {
				if time.Since(v.since) > time.Minute {
					delete(a.rates, k)
				}
			}
		}
		a.rateMu.Unlock()
		if limited {
			w.Header().Set("Retry-After", "60")
			problem(w, 429, "Too many requests")
			return
		}
		if a.Config.BasePath != "" && (r.URL.Path == a.Config.BasePath || strings.HasPrefix(r.URL.Path, a.Config.BasePath+"/")) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, a.Config.BasePath)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}
		defer func() {
			if recover() != nil {
				a.Logger.Print(`{"event":"request_panic"}`)
				problem(w, 500, "Request failed")
			}
		}()
		m.ServeHTTP(w, r)
	})
}
func (a *App) Static(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		problem(w, 405, "Method not allowed")
		return
	}
	if a.Assets == nil {
		problem(w, 503, "Frontend unavailable")
		return
	}
	p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if p == "." || p == "" {
		p = "index.html"
	}
	b, e := fs.ReadFile(a.Assets, p)
	if e != nil {
		if path.Ext(p) != "" {
			http.NotFound(w, r)
			return
		}
		p = "index.html"
		b, e = fs.ReadFile(a.Assets, p)
	}
	if e != nil {
		http.NotFound(w, r)
		return
	}
	if p == "index.html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		b = []byte(strings.ReplaceAll(string(b), "<!--MYTUBE_BASE-->", `<meta name="mytube-base" content="`+html.EscapeString(a.Config.BasePath+"/")+`">`))
		w.Write(b)
		return
	}
	http.ServeContent(w, r, p, time.Time{}, strings.NewReader(string(b)))
}
func (a *App) Me(w http.ResponseWriter, r *http.Request) {
	id := who(r)
	var channel string
	var personalize bool = true
	if id.User.ID != "" {
		a.Store.DB.QueryRow("SELECT id FROM channels WHERE owner=?", id.User.ID).Scan(&channel)
		a.Store.DB.QueryRow("SELECT personalize FROM users WHERE id=?", id.User.ID).Scan(&personalize)
	}
	respond(w, 200, map[string]any{"user": id.User, "csrf": id.CSRF, "channel_id": channel, "personalize": personalize, "version": Version, "mode": a.Config.Mode, "setup": a.Config.Setup, "upload_mb": a.Config.UploadMB, "public_uploads": a.Config.PublicUploads, "drive": a.Config.DriveURL != "", "ffmpeg": a.helper(a.Config.FFmpeg, "ffmpeg") != "", "ffprobe": a.helper(a.Config.FFprobe, "ffprobe") != "", "transcode": a.Config.Transcode, "streaming": a.helperProgress()})
}
func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
	a.Store.DB.Exec("DELETE FROM sessions WHERE hash=?", who(r).Hash)
	a.cookie(w, "mytube_session", "", -1)
	respond(w, 200, map[string]bool{"ok": true})
}
