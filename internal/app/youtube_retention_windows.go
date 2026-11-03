//go:build windows

package app

import "golang.org/x/sys/windows"

func youtubeReparse(path string) bool {
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return true
	}
	attributes, e := windows.GetFileAttributes(p)
	return e != nil || attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
