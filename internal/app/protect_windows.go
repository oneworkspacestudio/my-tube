//go:build windows

package app

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
)

// ProtectData applies a protected inheritable ACL to application data. Managed
// multi-user mode refuses storage without persistent ACL support.
func ProtectData(dir string, required bool) error {
	abs, e := filepath.Abs(dir)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(abs, 0700); e != nil {
		return e
	}
	root, e := windows.UTF16PtrFromString(filepath.VolumeName(abs) + `\`)
	if e != nil {
		return e
	}
	var flags uint32
	if e = windows.GetVolumeInformation(root, nil, 0, nil, nil, &flags, nil, 0); e != nil {
		return e
	}
	if flags&windows.FILE_PERSISTENT_ACLS == 0 {
		if required {
			return errors.New("managed data requires an ACL-capable filesystem such as NTFS")
		}
		return nil
	}
	tokenUser, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return e
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;" + tokenUser.User.Sid.String() + ")")
	if e != nil {
		return e
	}
	dacl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	// Apply to existing children too; a pre-existing explicit Everyone ACE must not
	// survive merely because the new root ACL is protected.
	return filepath.WalkDir(abs, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("data directory may not contain symlinks")
		}
		return windows.SetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
	})
}
