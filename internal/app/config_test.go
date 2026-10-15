package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceConfigurationAndUnsafeOrigins(t *testing.T) {
	t.Setenv("WORKSPACE_OIDC_ISSUER", "http://localhost:8000")
	t.Setenv("WORKSPACE_OIDC_CLIENT_ID", "tube-test")
	t.Setenv("WORKSPACE_OIDC_REDIRECT_URI", "http://localhost:9000/my-tube/auth/callback")
	c, e := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	if c.Setup || c.PublicURL != "http://localhost:9000/my-tube" || c.BasePath != "/my-tube" {
		t.Fatalf("%+v", c)
	}
	c.PublicURL = "http://localhost.evil.invalid"
	if c.Validate() == nil {
		t.Fatal("Nonloopback HTTP permitted")
	}
	c.PublicURL = "https://safe.example"
	c.RedirectURI = ""
	c.Mode = "local"
	if c.Validate() == nil {
		t.Fatal("Remote local mode permitted")
	}
}
func TestDedicatedDirectoriesAndFormatLimits(t *testing.T) {
	root := t.TempDir()
	if e := os.WriteFile(filepath.Join(root, "unrelated.txt"), []byte("existing"), 0600); e != nil {
		t.Fatal(e)
	}
	if CheckDataDirectory(root) == nil || CheckMediaDirectory(root) == nil {
		t.Fatal("Unrelated directories accepted")
	}
	c, e := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	c.AllowedFormats = []string{".exe"}
	if c.Validate() == nil {
		t.Fatal("Invalid format accepted")
	}
	c.AllowedFormats = []string{".mp4"}
	c.ProcessingHoursUTC = []int{25}
	if c.Validate() == nil {
		t.Fatal("Invalid idle hours accepted")
	}
}
