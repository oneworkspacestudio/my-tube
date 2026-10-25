//go:build !windows && !linux

package app

import "os"

func ProtectData(dir string, required bool) error {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	return os.Chmod(dir, 0700)
}
