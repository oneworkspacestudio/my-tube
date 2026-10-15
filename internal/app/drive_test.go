package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuantumDocumentedAPIAndCredentialScope(t *testing.T) {
	var exported bytes.Buffer
	drive := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer scoped.user.token" {
			w.WriteHeader(403)
			return
		}
		switch r.URL.Path {
		case "/api/users":
			respond(w, 200, map[string]any{"username": "alice", "permissions": map[string]bool{"admin": false}})
		case "/api/resources":
			if r.URL.Query().Get("source") != "videos" || r.URL.Query().Get("path") != "/clip.mp4" {
				w.WriteHeader(403)
				return
			}
			if r.Method == "POST" {
				if r.URL.Query().Get("override") != "false" {
					t.Error("Export must not overwrite")
				}
				io.Copy(&exported, r.Body)
				w.WriteHeader(201)
			} else {
				respond(w, 200, map[string]any{"name": "clip.mp4", "size": 5})
			}
		case "/api/resources/download":
			if r.URL.Query().Get("file") != "/clip.mp4" {
				w.WriteHeader(403)
				return
			}
			w.Write([]byte("video"))
		default:
			w.WriteHeader(404)
		}
	}))
	defer drive.Close()
	a := testApp(t)
	a.Config.DriveURL = drive.URL
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Drive-Token", "scoped.user.token")
	s, e := a.drive(r)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.List(context.Background(), "videos", "/clip.mp4"); e != nil {
		t.Fatal(e)
	}
	b, _, e := s.Read(context.Background(), "videos", "/clip.mp4")
	if e != nil {
		t.Fatal(e)
	}
	v, _ := io.ReadAll(b)
	b.Close()
	if string(v) != "video" {
		t.Fatal("Wrong download")
	}
	if e = s.Write(context.Background(), "videos", "/clip.mp4", bytes.NewReader(v), 5); e != nil {
		t.Fatal(e)
	}
	if exported.String() != "video" {
		t.Fatal("Wrong upload")
	}
	if _, e = s.List(context.Background(), "videos", "/forbidden"); e == nil {
		t.Fatal("Drive ACL denial ignored")
	}
	if drivePath(driveInput{Source: "videos", Path: "/../secret"}) {
		t.Fatal("Traversal allowed")
	}
}
func TestQuantumRejectsAdminAndRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("Drive token leaked through redirect") }))
	defer target.Close()
	drive := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer admin.token.jwt" {
			respond(w, 200, map[string]any{"username": "admin", "permissions": map[string]bool{"admin": true}})
		} else {
			http.Redirect(w, r, target.URL, 302)
		}
	}))
	defer drive.Close()
	a := testApp(t)
	a.Config.DriveURL = drive.URL
	for _, token := range []string{"admin.token.jwt", "redirect.token.jwt"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Drive-Token", token)
		if _, e := a.drive(r); e == nil {
			t.Fatal("Unsafe Drive credential accepted")
		}
	}
}
