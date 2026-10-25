//go:build linux

package app

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func ProtectData(dir string, required bool) error {
	abs, e := filepath.Abs(dir)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(abs, 0700); e != nil {
		return e
	}
	for p := abs; ; p = filepath.Dir(p) {
		st, e := os.Lstat(p)
		if e != nil {
			return e
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return errors.New("private app data must not use symbolic links")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return filepath.WalkDir(abs, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		st, e := d.Info()
		if e != nil {
			return e
		}
		if !d.IsDir() && !st.Mode().IsRegular() {
			return errors.New("private app data must contain ordinary files and directories")
		}
		raw, ok := st.Sys().(*syscall.Stat_t)
		if !ok || raw.Uid != uint32(os.Geteuid()) || (!d.IsDir() && raw.Nlink != 1) {
			return errors.New("private app data belongs to another user or contains hard links")
		}
		mode := os.FileMode(0600)
		if d.IsDir() {
			mode = 0700
		}
		if e = os.Chmod(p, mode); e != nil {
			return e
		}
		check, e := os.Stat(p)
		if e != nil {
			return e
		}
		if check.Mode().Perm() != mode {
			return errors.New("private app data requires a Linux filesystem with Unix permissions")
		}
		return nil
	})
}
