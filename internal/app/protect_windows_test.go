//go:build windows

package app

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDataACLReplacesExistingEveryoneGrant(t *testing.T) {
	if root := os.Getenv("MYTUBE_ACL_TEST_ROOT"); root != "" {
		t.Setenv("GOTMPDIR", root)
		t.Setenv("TEMP", root)
		t.Setenv("TMP", root)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "private.txt")
	if e := os.WriteFile(p, []byte("private"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := ProtectData(dir, true); e != nil {
		t.Skipf("test filesystem does not support ACLs: %v", e)
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;FA;;;WD)")
	if e != nil {
		t.Fatal(e)
	}
	dacl, _, e := sd.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); e != nil {
		t.Fatal(e)
	}
	if e = ProtectData(dir, true); e != nil {
		t.Fatal(e)
	}
	current, e := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(current.String(), ";;;WD)") {
		t.Fatal("Everyone grant survived", current.String())
	}
	if !strings.Contains(current.String(), "D:P") {
		t.Fatal("ACL is not protected", current.String())
	}
}
