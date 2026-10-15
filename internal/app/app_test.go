package app

import (
	"bytes"
	"encoding/json"

	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func testApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	c, e := LoadConfig(filepath.Join(dir, "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	c.Mode = "managed"
	c.Setup = false
	c.MediaDir = filepath.Join(dir, "media")
	c.Issuer = "http://localhost:9999"
	c.PublicUploads = true
	c.StorageMB = 100
	c.QuotaMB = 50
	c.UploadMB = 10
	a, e := New(c, fstest.MapFS{"index.html": {Data: []byte("<!--MYTUBE_BASE--><h1>My Tube</h1>")}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { a.Close() })
	return a
}

type session struct {
	User   User
	Cookie *http.Cookie
	CSRF   string
}

func login(t *testing.T, a *App, sub string) session {
	t.Helper()
	u, e := a.Store.Provision(a.Config.Issuer, sub, sub, sub+"@test.invalid", "", 50)
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	if e = a.newSession(w, u); e != nil {
		t.Fatal(e)
	}
	s := session{User: u, Cookie: w.Result().Cookies()[0]}
	a.Store.DB.QueryRow("SELECT csrf FROM sessions WHERE hash=?", digest(s.Cookie.Value)).Scan(&s.CSRF)
	return s
}
func call(t *testing.T, a *App, s session, method, p string, body any, extra map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var b io.Reader
	if raw, ok := body.([]byte); ok {
		b = bytes.NewReader(raw)
	} else if body != nil {
		v, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		b = bytes.NewReader(v)
	}
	r := httptest.NewRequest(method, a.Config.PublicURL+p, b)
	r.Header.Set("Origin", a.Config.PublicURL)
	r.Header.Set("X-CSRF-Token", s.CSRF)
	if s.Cookie != nil {
		r.AddCookie(s.Cookie)
	}
	for k, v := range extra {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	return w
}
func mustStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP %d expected %d: %s", w.Code, status, w.Body.String())
	}
}
func result(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	return v
}
func channel(t *testing.T, a *App, s session) string {
	t.Helper()
	w := call(t, a, s, "POST", "/api/channel", map[string]any{"name": s.User.Name + " Channel", "handle": s.User.Name + "-channel", "visibility": "public"}, nil)
	mustStatus(t, w, 200)
	return result(t, w)["id"].(string)
}
func uploadVideo(t *testing.T, a *App, s session, vis string) string {
	t.Helper()
	b, e := os.ReadFile("testdata/sample.mp4")
	if e != nil {
		t.Fatal(e)
	}
	w := call(t, a, s, "POST", "/api/uploads", map[string]any{"name": "sample.mp4", "size": len(b), "title": "Ocean technology", "visibility": vis}, nil)
	mustStatus(t, w, 201)
	id := result(t, w)["id"].(string)
	w = call(t, a, s, "PATCH", "/api/uploads/"+id, b, map[string]string{"Upload-Offset": "0"})
	mustStatus(t, w, 200)
	w = call(t, a, s, "POST", "/api/uploads/"+id+"/complete", map[string]any{}, nil)
	mustStatus(t, w, 201)
	return result(t, w)["id"].(string)
}
func TestUploadPlaybackVisibilityAndAllDerivedRoutes(t *testing.T) {
	a := testApp(t)
	owner := login(t, a, "alice")
	other := login(t, a, "bob")
	channel(t, a, owner)
	id := uploadVideo(t, a, owner, "private")
	for _, p := range []string{"/api/videos/" + id, "/api/videos/" + id + "/comments", "/api/videos/" + id + "/media/original", "/api/videos/" + id + "/media/poster.jpg", "/api/videos/" + id + "/media/master.m3u8", "/api/videos/" + id + "/media/q360-00000.ts", "/api/videos/" + id + "/media/captions.vtt", "/api/videos/" + id + "/media/original?download=1"} {
		for _, s := range []session{{}, other} {
			mustStatus(t, call(t, a, s, "GET", p, nil, nil), 404)
		}
	}
	w := call(t, a, owner, "GET", "/api/videos/"+id+"/media/original", nil, map[string]string{"Range": "bytes=0-31"})
	mustStatus(t, w, 206)
	if w.Body.Len() != 32 || !strings.HasPrefix(w.Header().Get("Content-Range"), "bytes 0-31/") {
		t.Fatal("Range contract")
	}
	mustStatus(t, call(t, a, other, "PUT", "/api/videos/"+id, map[string]any{"title": "stolen"}, nil), 404)
	mustStatus(t, call(t, a, owner, "PUT", "/api/videos/"+id+"/access", map[string]any{"subject": "bob", "allow": true}, nil), 200)
	mustStatus(t, call(t, a, other, "GET", "/api/videos/"+id, nil, nil), 200)
	edit := map[string]any{"title": "Ocean technology", "visibility": "unlisted", "category": "Nature", "chapters": []any{}}
	mustStatus(t, call(t, a, owner, "PUT", "/api/videos/"+id, edit, nil), 200)
	mustStatus(t, call(t, a, session{}, "GET", "/api/videos/"+id, nil, nil), 404)
	a.Config.PublicAccess = true
	mustStatus(t, call(t, a, session{}, "GET", "/api/videos/"+id, nil, nil), 200)
	w = call(t, a, other, "GET", "/api/videos?q=Ocean", nil, nil)
	mustStatus(t, w, 200)
	if w.Body.String() != "[]\n" {
		t.Fatal("Unlisted discovery leak")
	}
	edit["visibility"] = "public"
	mustStatus(t, call(t, a, owner, "PUT", "/api/videos/"+id, edit, nil), 200)
	w = call(t, a, session{}, "GET", "/api/videos?q=Ocean", nil, nil)
	mustStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), id) {
		t.Fatal("FTS not indexed")
	}
	mustStatus(t, call(t, a, owner, "DELETE", "/api/videos/"+id, nil, nil), 200)
	mustStatus(t, call(t, a, owner, "GET", "/api/videos/"+id+"/media/original", nil, nil), 404)
	if _, e := os.Stat(filepath.Join(a.Config.MediaDir, "originals", id+".mp4")); e != nil {
		t.Fatal("Original retention failed", e)
	}
}
func TestInterruptedUploadQuotaValidationAndCSRF(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	other := login(t, a, "bob")
	channel(t, a, s)
	b, _ := os.ReadFile("testdata/sample.mp4")
	w := call(t, a, s, "POST", "/api/uploads", map[string]any{"name": "resume.mp4", "size": len(b), "visibility": "private"}, nil)
	mustStatus(t, w, 201)
	id := result(t, w)["id"].(string)
	mustStatus(t, call(t, a, other, "PATCH", "/api/uploads/"+id, b[:100], map[string]string{"Upload-Offset": "0"}), 404)
	mustStatus(t, call(t, a, s, "PATCH", "/api/uploads/"+id, b[:100], map[string]string{"Upload-Offset": "0"}), 200)
	mustStatus(t, call(t, a, s, "PATCH", "/api/uploads/"+id, b[:100], map[string]string{"Upload-Offset": "0"}), 409)
	mustStatus(t, call(t, a, s, "POST", "/api/uploads/"+id+"/complete", nil, nil), 400)
	f, e := os.OpenFile(a.incoming(id), os.O_WRONLY|os.O_APPEND, 0600)
	if e != nil {
		t.Fatal(e)
	}
	f.Write([]byte("unacknowledged"))
	f.Close()
	if e = a.Recover(); e != nil {
		t.Fatal(e)
	}
	st, _ := os.Stat(a.incoming(id))
	if st.Size() != 100 {
		t.Fatal("Recovery must discard unacknowledged bytes")
	}
	mustStatus(t, call(t, a, s, "PATCH", "/api/uploads/"+id, b[100:], map[string]string{"Upload-Offset": "100"}), 200)
	mustStatus(t, call(t, a, s, "POST", "/api/uploads/"+id+"/complete", nil, nil), 201)
	mustStatus(t, call(t, a, s, "POST", "/api/uploads", map[string]any{"name": "../evil.mp4", "size": 10, "visibility": "private"}, nil), 400)
	mustStatus(t, call(t, a, s, "POST", "/api/uploads", map[string]any{"name": "big.mp4", "size": 11 << 20, "visibility": "private"}, nil), 400)
	mustStatus(t, call(t, a, s, "POST", "/api/channel", map[string]any{}, map[string]string{"Origin": "https://attacker.invalid"}), 403)
	w = call(t, a, s, "POST", "/api/uploads", map[string]any{"name": "fake.mp4", "size": 50, "visibility": "private"}, nil)
	mustStatus(t, w, 201)
	fake := result(t, w)["id"].(string)
	mustStatus(t, call(t, a, s, "PATCH", "/api/uploads/"+fake, bytes.Repeat([]byte("x"), 50), map[string]string{"Upload-Offset": "0"}), 200)
	mustStatus(t, call(t, a, s, "POST", "/api/uploads/"+fake+"/complete", nil, nil), 400)
	w = call(t, a, s, "POST", "/api/uploads", map[string]any{"name": "duplicate.mp4", "size": len(b), "visibility": "private"}, nil)
	mustStatus(t, w, 201)
	dup := result(t, w)["id"].(string)
	mustStatus(t, call(t, a, s, "PATCH", "/api/uploads/"+dup, b, map[string]string{"Upload-Offset": "0"}), 200)
	mustStatus(t, call(t, a, s, "POST", "/api/uploads/"+dup+"/complete", nil, nil), 400)
}
func TestSocialSearchPlaylistsAndOwnership(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	b := login(t, a, "bob")
	ch := channel(t, a, s)
	mustStatus(t, call(t, a, b, "POST", "/api/channels/"+ch+"/subscribe", map[string]any{"enabled": true}, nil), 200)
	id := uploadVideo(t, a, s, "public")
	for _, kind := range []string{"like", "later"} {
		mustStatus(t, call(t, a, b, "POST", "/api/videos/"+id+"/"+kind, map[string]any{"enabled": true}, nil), 200)
	}
	w := call(t, a, b, "GET", "/api/notifications", nil, nil)
	mustStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), id) {
		t.Fatal("Missing upload notification")
	}
	w = call(t, a, b, "POST", "/api/videos/"+id+"/comments", map[string]any{"body": "<script>alert('escaped')</script> Great video"}, nil)
	mustStatus(t, w, 201)
	cid := result(t, w)["id"].(string)
	mustStatus(t, call(t, a, s, "PUT", "/api/comments/"+cid, map[string]any{"body": "changed"}, nil), 404)
	mustStatus(t, call(t, a, b, "PUT", "/api/comments/"+cid, map[string]any{"body": "Updated comment"}, nil), 200)
	mustStatus(t, call(t, a, s, "POST", "/api/videos/"+id+"/comments", map[string]any{"parent": cid, "body": "Thanks"}, nil), 201)
	w = call(t, a, b, "POST", "/api/playlists", map[string]any{"title": "Weekend technology", "description": "Ocean", "visibility": "private"}, nil)
	mustStatus(t, w, 200)
	pl := result(t, w)["id"].(string)
	mustStatus(t, call(t, a, b, "POST", "/api/playlists/"+pl+"/videos", map[string]any{"video_id": id}, nil), 200)
	mustStatus(t, call(t, a, s, "GET", "/api/videos?playlist="+pl, nil, nil), 404)
	mustStatus(t, call(t, a, b, "GET", "/api/videos?playlist="+pl, nil, nil), 200)
	mustStatus(t, call(t, a, s, "DELETE", "/api/playlists/"+pl, nil, nil), 404)
	mustStatus(t, call(t, a, b, "POST", "/api/videos/"+id+"/progress", map[string]any{"position": 1.5, "seconds": 0}, nil), 200)
	w = call(t, a, b, "GET", "/api/videos?view=history", nil, nil)
	if !strings.Contains(w.Body.String(), id) {
		t.Fatal("No resume history")
	}
	mustStatus(t, call(t, a, s, "DELETE", "/api/comments/"+cid, nil, nil), 200)
	w = call(t, a, b, "GET", "/api/playlists?q=Weekend", nil, nil)
	mustStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), pl) {
		t.Fatal("Playlist FTS missing")
	}
	mustStatus(t, call(t, a, b, "POST", "/api/logout", nil, nil), 200)
	mustStatus(t, call(t, a, b, "GET", "/api/studio", nil, nil), 401)
}
func TestBackupRestoreAndMigrationGuards(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	channel(t, a, s)
	id := uploadVideo(t, a, s, "private")
	archive := filepath.Join(t.TempDir(), "backup.zip")
	if e := BackupAll(a.Config, archive); e != nil {
		t.Fatal(e)
	}
	c := a.Config
	c.DataDir = filepath.Join(t.TempDir(), "restored")
	c.MediaDir = filepath.Join(c.DataDir, "media")
	if e := RestoreAll(c, archive); e != nil {
		t.Fatal(e)
	}
	db, e := OpenStore(c.DataDir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var n int
	db.DB.QueryRow("SELECT count(*) FROM videos WHERE id=?", id).Scan(&n)
	if n != 1 {
		t.Fatal("Video relationship lost")
	}
	db.DB.QueryRow("SELECT count(*) FROM sessions").Scan(&n)
	if n != 0 {
		t.Fatal("Restored sessions must be invalidated")
	}
	if _, e = os.Stat(filepath.Join(c.MediaDir, "originals", id+".mp4")); e != nil {
		t.Fatal(e)
	}
	if e = RestoreAll(c, archive); e == nil {
		t.Fatal("Restore must refuse overwrite")
	}
	db.DB.Exec("UPDATE migrations SET version=99")
	if _, e = OpenStore(c.DataDir); e == nil {
		t.Fatal("Downgrade allowed")
	}
}
func TestMP4ValidationAndPrefixAssets(t *testing.T) {
	m, e := inspectMP4("testdata/sample.mp4")
	if e != nil || !m.Direct || m.Width != 640 || m.Height != 360 || m.Duration < 2.9 {
		t.Fatalf("%+v %v", m, e)
	}
	a := testApp(t)
	a.Config.PublicURL += "/my-tube"
	a.Config.BasePath = "/my-tube"
	w := call(t, a, session{}, "GET", "/", nil, nil)
	mustStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), `content="/my-tube/"`) {
		t.Fatal("Missing prefix")
	}
	bad := filepath.Join(t.TempDir(), "fake.mp4")
	os.WriteFile(bad, []byte("0000ftypgarbage"), 0600)
	if _, e = inspectMP4(bad); e == nil {
		t.Fatal("Invalid BMFF accepted")
	}
}
func TestFirstRunLocalLinkIsSingleUse(t *testing.T) {
	a := testApp(t)
	a.Config.Mode = "local"
	u := a.LocalURL()
	r := httptest.NewRequest("GET", u, nil)
	r.RemoteAddr = "127.0.0.1:50000"
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	mustStatus(t, w, 303)
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	mustStatus(t, w, 401)
	a.Config.Mode = "managed"
	a.Config.Setup = false
	r = httptest.NewRequest("GET", a.LocalURL(), nil)
	r.RemoteAddr = "127.0.0.1:50000"
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	mustStatus(t, w, 401)
}

func TestReplacementPublishesAtomicallyWithNewURL(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	channel(t, a, s)
	old := uploadVideo(t, a, s, "public")
	raw, _ := os.ReadFile("testdata/sample.mp4")
	raw = append(raw, 0, 0, 0, 8, 'f', 'r', 'e', 'e')
	w := call(t, a, s, "POST", "/api/uploads", map[string]any{"name": "replacement.mp4", "size": len(raw), "title": "Replacement", "visibility": "private"}, nil)
	mustStatus(t, w, 201)
	id := result(t, w)["id"].(string)
	mustStatus(t, call(t, a, s, "PATCH", "/api/uploads/"+id, raw, map[string]string{"Upload-Offset": "0"}), 200)
	mustStatus(t, call(t, a, s, "POST", "/api/uploads/"+id+"/complete", nil, nil), 201)
	mustStatus(t, call(t, a, s, "POST", "/api/videos/"+old+"/replace", map[string]any{"replacement_id": id}, nil), 200)
	mustStatus(t, call(t, a, s, "GET", "/api/videos/"+old, nil, nil), 404)
	w = call(t, a, s, "GET", "/api/videos/"+id, nil, nil)
	mustStatus(t, w, 200)
	v := result(t, w)
	if v["title"] != "Ocean technology" || v["visibility"] != "public" {
		t.Fatal(v)
	}
	if _, e := os.Stat(filepath.Join(a.Config.MediaDir, "originals", old+".mp4")); e != nil {
		t.Fatal("Old original lost", e)
	}
}
