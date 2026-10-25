package app

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

func ID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func secret() string         { return ID() + ID() }
func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func now() int64             { return time.Now().UnixMilli() }

type User struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Avatar  string `json:"avatar"`
	Admin   bool   `json:"admin"`
	Subject string `json:"-"`
}
type Store struct {
	DB  *sql.DB
	Dir string
}

const schema = `
CREATE TABLE migrations(version INTEGER PRIMARY KEY,applied INTEGER NOT NULL);
CREATE TABLE users(id TEXT PRIMARY KEY,issuer TEXT NOT NULL,subject TEXT NOT NULL,name TEXT NOT NULL,email TEXT NOT NULL,avatar TEXT NOT NULL,quota_mb INTEGER NOT NULL,disabled INTEGER NOT NULL DEFAULT 0,personalize INTEGER NOT NULL DEFAULT 1,UNIQUE(issuer,subject));
CREATE TABLE sessions(hash TEXT PRIMARY KEY,user_id TEXT NOT NULL REFERENCES users(id),csrf TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE TABLE logins(state TEXT PRIMARY KEY,binding TEXT NOT NULL,nonce TEXT NOT NULL,verifier TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE TABLE channels(id TEXT PRIMARY KEY,owner TEXT NOT NULL UNIQUE REFERENCES users(id),name TEXT NOT NULL,handle TEXT NOT NULL UNIQUE COLLATE NOCASE,description TEXT NOT NULL DEFAULT '',avatar TEXT NOT NULL DEFAULT '',banner TEXT NOT NULL DEFAULT '',visibility TEXT NOT NULL DEFAULT 'public',created INTEGER NOT NULL);
CREATE TABLE videos(id TEXT PRIMARY KEY,owner TEXT NOT NULL REFERENCES users(id),channel_id TEXT NOT NULL REFERENCES channels(id),title TEXT NOT NULL,description TEXT NOT NULL DEFAULT '',tags TEXT NOT NULL DEFAULT '',category TEXT NOT NULL DEFAULT 'General',visibility TEXT NOT NULL CHECK(visibility IN('private','unlisted','public')),original TEXT NOT NULL,filename TEXT NOT NULL,size INTEGER NOT NULL,sha256 TEXT NOT NULL,duration REAL NOT NULL DEFAULT 0,width INTEGER NOT NULL DEFAULT 0,height INTEGER NOT NULL DEFAULT 0,direct INTEGER NOT NULL DEFAULT 0,status TEXT NOT NULL DEFAULT 'queued',error TEXT NOT NULL DEFAULT '',created INTEGER NOT NULL,views INTEGER NOT NULL DEFAULT 0,watch_seconds REAL NOT NULL DEFAULT 0,chapters TEXT NOT NULL DEFAULT '[]',deleted INTEGER NOT NULL DEFAULT 0,UNIQUE(owner,sha256));
CREATE INDEX videos_channel ON videos(channel_id,created DESC);
CREATE INDEX videos_discovery ON videos(visibility,deleted,created DESC);
CREATE VIRTUAL TABLE video_search USING fts5(title,description,tags,channel,category);
CREATE TRIGGER video_delete AFTER DELETE ON videos BEGIN DELETE FROM video_search WHERE rowid=old.rowid;END;
CREATE TABLE video_access(video_id TEXT REFERENCES videos(id) ON DELETE CASCADE,user_id TEXT REFERENCES users(id),PRIMARY KEY(video_id,user_id));
CREATE TABLE uploads(id TEXT PRIMARY KEY,owner TEXT NOT NULL REFERENCES users(id),channel_id TEXT NOT NULL,name TEXT NOT NULL,title TEXT NOT NULL,visibility TEXT NOT NULL,size INTEGER NOT NULL,offset INTEGER NOT NULL DEFAULT 0,created INTEGER NOT NULL,video_id TEXT NOT NULL DEFAULT '');
CREATE TABLE jobs(id TEXT PRIMARY KEY,video_id TEXT NOT NULL UNIQUE REFERENCES videos(id) ON DELETE CASCADE,status TEXT NOT NULL,attempts INTEGER NOT NULL DEFAULT 0,error TEXT NOT NULL DEFAULT '',updated INTEGER NOT NULL);
CREATE TABLE subscriptions(user_id TEXT REFERENCES users(id),channel_id TEXT REFERENCES channels(id) ON DELETE CASCADE,created INTEGER NOT NULL,PRIMARY KEY(user_id,channel_id));
CREATE TABLE likes(user_id TEXT REFERENCES users(id),video_id TEXT REFERENCES videos(id) ON DELETE CASCADE,PRIMARY KEY(user_id,video_id));
CREATE TABLE later(user_id TEXT REFERENCES users(id),video_id TEXT REFERENCES videos(id) ON DELETE CASCADE,PRIMARY KEY(user_id,video_id));
CREATE TABLE history(user_id TEXT REFERENCES users(id),video_id TEXT REFERENCES videos(id) ON DELETE CASCADE,position REAL NOT NULL DEFAULT 0,seconds REAL NOT NULL DEFAULT 0,updated INTEGER NOT NULL,counted INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(user_id,video_id));
CREATE TABLE playlists(id TEXT PRIMARY KEY,owner TEXT NOT NULL REFERENCES users(id),title TEXT NOT NULL,description TEXT NOT NULL DEFAULT '',visibility TEXT NOT NULL DEFAULT 'private',created INTEGER NOT NULL);
CREATE VIRTUAL TABLE playlist_search USING fts5(title,description);
CREATE TABLE playlist_videos(playlist_id TEXT REFERENCES playlists(id) ON DELETE CASCADE,video_id TEXT REFERENCES videos(id) ON DELETE CASCADE,position INTEGER NOT NULL,PRIMARY KEY(playlist_id,video_id));
CREATE TABLE comments(id TEXT PRIMARY KEY,video_id TEXT REFERENCES videos(id) ON DELETE CASCADE,user_id TEXT REFERENCES users(id),parent TEXT NOT NULL DEFAULT '',body TEXT NOT NULL,created INTEGER NOT NULL,updated INTEGER NOT NULL,hidden INTEGER NOT NULL DEFAULT 0);
CREATE INDEX comments_video ON comments(video_id,created);
CREATE TABLE notifications(id TEXT PRIMARY KEY,user_id TEXT REFERENCES users(id),video_id TEXT REFERENCES videos(id) ON DELETE CASCADE,body TEXT NOT NULL,created INTEGER NOT NULL,seen INTEGER NOT NULL DEFAULT 0);
CREATE TABLE reports(id TEXT PRIMARY KEY,user_id TEXT REFERENCES users(id),video_id TEXT REFERENCES videos(id) ON DELETE CASCADE,reason TEXT NOT NULL,status TEXT NOT NULL DEFAULT 'open',created INTEGER NOT NULL);
CREATE TABLE analytics(day TEXT NOT NULL,channel_id TEXT REFERENCES channels(id) ON DELETE CASCADE,subscribers INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(day,channel_id));
CREATE TABLE audit(id INTEGER PRIMARY KEY,at INTEGER NOT NULL,actor TEXT NOT NULL,event TEXT NOT NULL,target TEXT NOT NULL);
INSERT INTO migrations VALUES(1,0);
PRAGMA application_id=1297372482;
`

func OpenStore(dir string) (*Store, error) {
	for _, p := range []string{"uploads", "backups", "logs"} {
		if e := os.MkdirAll(filepath.Join(dir, p), 0700); e != nil {
			return nil, e
		}
	}
	db, e := sql.Open("sqlite", filepath.Join(dir, "mytube.db"))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	if _, e = db.Exec("PRAGMA journal_mode=WAL;PRAGMA synchronous=FULL;PRAGMA foreign_keys=ON;PRAGMA busy_timeout=5000"); e != nil {
		return fail(e)
	}
	var count int
	if e = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&count); e != nil {
		return fail(e)
	}
	if count == 0 {
		var n int
		db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&n)
		if n != 0 {
			return fail(errors.New("unrelated database"))
		}
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		if _, e = tx.Exec(schema); e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	} else {
		var v, app int
		db.QueryRow("SELECT MAX(version) FROM migrations").Scan(&v)
		db.QueryRow("PRAGMA application_id").Scan(&app)
		if (v != 1 && v != 2) || app != 1297372482 {
			return fail(errors.New("unsupported database version or application"))
		}
	}
	if _, e = db.Exec(youtubeSchema); e != nil {
		return fail(e)
	}
	return &Store{db, dir}, nil
}
func (s *Store) Close() error {
	_, e := s.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return errors.Join(e, s.DB.Close())
}
func (s *Store) Provision(issuer, sub, name, email, avatar string, quota int64) (User, error) {
	if name == "" {
		name = "Workspace member"
	}
	id := ID()
	_, e := s.DB.Exec(`INSERT INTO users(id,issuer,subject,name,email,avatar,quota_mb)VALUES(?,?,?,?,?,?,?) ON CONFLICT(issuer,subject) DO UPDATE SET name=excluded.name,email=excluded.email,avatar=excluded.avatar`, id, issuer, sub, name, email, avatar, quota)
	if e != nil {
		return User{}, e
	}
	var u User
	var disabled bool
	e = s.DB.QueryRow("SELECT id,name,email,avatar,subject,disabled FROM users WHERE issuer=? AND subject=?", issuer, sub).Scan(&u.ID, &u.Name, &u.Email, &u.Avatar, &u.Subject, &disabled)
	if disabled {
		return u, errors.New("account disabled")
	}
	return u, e
}
func (s *Store) Audit(actor, event, target string) {
	s.DB.Exec("INSERT INTO audit(at,actor,event,target)VALUES(?,?,?,?)", now(), actor, event, target)
}

type scanner interface{ Scan(...any) error }
