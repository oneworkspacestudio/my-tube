package app

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type backupManifest struct {
	Version int               `json:"version"`
	Files   map[string]string `json:"files"`
}

var objectID = regexp.MustCompile(`^[a-f0-9]{32}$`)
var originalKey = regexp.MustCompile(`^[a-f0-9]{32}\.(mp4|mov|mkv|webm)$`)

func verifyRelationships(s *Store, media string) error {
	rows, e := s.DB.Query("SELECT id,original,size FROM videos")
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var id, key string
		var size int64
		if e = rows.Scan(&id, &key, &size); e != nil {
			return e
		}
		if !objectID.MatchString(id) || !originalKey.MatchString(key) {
			return errors.New("Invalid media relationship")
		}
		st, e := os.Stat(filepath.Join(media, "originals", key))
		if e != nil || !st.Mode().IsRegular() || st.Size() != size {
			return fmt.Errorf("Original missing or size mismatch for video %s", id)
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	rows, e = s.DB.Query("SELECT id,name FROM uploads WHERE video_id=''")
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if e = rows.Scan(&id, &name); e != nil {
			return e
		}
		if !objectID.MatchString(id) || filepath.Base(name) != name || strings.ContainsAny(name, "\\/:\x00") {
			return errors.New("Invalid upload relationship")
		}
	}
	return rows.Err()
}

func BackupAll(c Config, out string) error {
	s, e := OpenStore(c.DataDir)
	if e != nil {
		return e
	}
	defer s.Close()
	if e = verifyRelationships(s, c.MediaDir); e != nil {
		return e
	}
	snapshot := filepath.Join(c.DataDir, "backups", "snapshot-"+ID()+".db")
	defer os.Remove(snapshot)
	if _, e = s.DB.Exec("VACUUM INTO ?", snapshot); e != nil {
		return e
	}
	dest, e := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	z := zip.NewWriter(dest)
	manifest := backupManifest{1, map[string]string{}}
	add := func(name, p string) error {
		f, e := os.Open(p)
		if e != nil {
			return e
		}
		defer f.Close()
		entry, e := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if e != nil {
			return e
		}
		hash := sha256.New()
		if _, e = io.Copy(io.MultiWriter(entry, hash), f); e != nil {
			return e
		}
		manifest.Files[name] = hex.EncodeToString(hash.Sum(nil))
		return nil
	}
	e = add("mytube.db", snapshot)
	if e == nil {
		e = filepath.WalkDir(c.MediaDir, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.Type()&os.ModeSymlink != 0 {
				return errors.New("Refusing symlink in media")
			}
			if d.IsDir() {
				return nil
			}
			rel, e := filepath.Rel(c.MediaDir, p)
			if e != nil {
				return e
			}
			return add("media/"+filepath.ToSlash(rel), p)
		})
	}
	if e == nil {
		entry, err := z.Create("manifest.json")
		e = err
		if e == nil {
			e = json.NewEncoder(entry).Encode(manifest)
		}
	}
	e = errors.Join(e, z.Close(), dest.Sync(), dest.Close())
	if e != nil {
		os.Remove(out)
	}
	return e
}
func RestoreAll(c Config, source string) error {
	if _, e := os.Stat(filepath.Join(c.DataDir, "mytube.db")); !os.IsNotExist(e) {
		return errors.New("Restore requires a fresh destination")
	}
	media, e := filepath.Abs(c.MediaDir)
	if e != nil {
		return e
	}
	expected, _ := filepath.Abs(filepath.Join(c.DataDir, "media"))
	if media != expected {
		return errors.New("Restore into a fresh dedicated data/media folder, then relocate it while stopped")
	}
	entries, _ := os.ReadDir(media)
	if len(entries) > 0 {
		return errors.New("Restore media folder must be empty")
	}
	z, e := zip.OpenReader(source)
	if e != nil {
		return e
	}
	defer z.Close()
	if len(z.File) > 1000000 {
		return errors.New("Backup contains too many entries")
	}
	var manifest backupManifest
	found := false
	for _, f := range z.File {
		if f.Name == "manifest.json" {
			r, e := f.Open()
			if e != nil {
				return e
			}
			e = json.NewDecoder(io.LimitReader(r, 32<<20)).Decode(&manifest)
			r.Close()
			if e != nil || manifest.Version != 1 {
				return errors.New("Invalid backup manifest")
			}
			found = true
		}
	}
	if !found || manifest.Files["mytube.db"] == "" {
		return errors.New("Backup manifest/database missing")
	}
	stage := filepath.Join(c.DataDir, "backups", "restore-"+ID())
	if e = os.MkdirAll(stage, 0700); e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	seen := map[string]bool{}
	var total uint64
	for _, f := range z.File {
		if f.Name == "manifest.json" {
			continue
		}
		name := f.Name
		if seen[name] || strings.ContainsAny(name, "\\:\x00") || name != path.Clean(name) || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "../") || name != "mytube.db" && !strings.HasPrefix(name, "media/") || manifest.Files[name] == "" || f.Mode()&os.ModeSymlink != 0 {
			return errors.New("Unsafe or unexpected backup path")
		}
		seen[name] = true
		total += f.UncompressedSize64
		if total > uint64(c.StorageMB<<20)+1<<30 {
			return errors.New("Backup exceeds configured capacity")
		}
		dest := filepath.Join(stage, filepath.FromSlash(name))
		if e = os.MkdirAll(filepath.Dir(dest), 0700); e != nil {
			return e
		}
		src, e := f.Open()
		if e != nil {
			return e
		}
		out, e := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			src.Close()
			return e
		}
		hash := sha256.New()
		_, e = io.Copy(io.MultiWriter(out, hash), io.LimitReader(src, int64(f.UncompressedSize64)+1))
		e = errors.Join(e, out.Sync(), out.Close(), src.Close())
		if e != nil {
			return e
		}
		if hex.EncodeToString(hash.Sum(nil)) != manifest.Files[name] {
			return fmt.Errorf("Backup integrity failed: %s", name)
		}
	}
	if len(seen) != len(manifest.Files) {
		return errors.New("Backup files missing")
	}
	s, e := OpenStore(stage)
	if e != nil {
		return e
	}
	if e = verifyRelationships(s, filepath.Join(stage, "media")); e != nil {
		s.Close()
		return e
	}
	var check string
	e = s.DB.QueryRow("PRAGMA integrity_check").Scan(&check)
	if e != nil || check != "ok" {
		s.Close()
		return errors.New("Database integrity check failed")
	}
	_, e = s.DB.Exec("DELETE FROM sessions;DELETE FROM logins")
	if ce := s.Close(); e != nil || ce != nil {
		return errors.Join(e, ce)
	}
	if e = os.MkdirAll(c.DataDir, 0700); e != nil {
		return e
	}
	if _, e = os.Stat(filepath.Join(stage, "media")); e == nil {
		if e = os.Rename(filepath.Join(stage, "media"), c.MediaDir); e != nil {
			return e
		}
	}
	return os.Rename(filepath.Join(stage, "mytube.db"), filepath.Join(c.DataDir, "mytube.db"))
}
