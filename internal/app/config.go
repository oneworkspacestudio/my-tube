package app

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const Version = "0.1.0-preview.1"

type Config struct {
	Listen             string   `json:"listen"`
	PublicURL          string   `json:"public_url"`
	DataDir            string   `json:"data_dir"`
	MediaDir           string   `json:"media_dir"`
	Mode               string   `json:"mode"`
	Issuer             string   `json:"oidc_issuer"`
	ClientID           string   `json:"oidc_client_id"`
	ClientSecret       string   `json:"-"`
	RedirectURI        string   `json:"-"`
	AdminSubjects      []string `json:"admin_subjects"`
	SessionHours       int      `json:"session_hours"`
	UploadMB           int64    `json:"upload_mb"`
	QuotaMB            int64    `json:"quota_mb"`
	StorageMB          int64    `json:"storage_mb"`
	PublicAccess       bool     `json:"public_access"`
	PublicUploads      bool     `json:"public_uploads"`
	Workers            int      `json:"workers"`
	Threads            int      `json:"threads"`
	JobMinutes         int      `json:"job_minutes"`
	Qualities          []int    `json:"qualities"`
	AllowedFormats     []string `json:"allowed_formats"`
	ProcessingHoursUTC []int    `json:"processing_hours_utc"`
	Transcode          bool     `json:"transcode"`
	FFmpeg             string   `json:"ffmpeg"`
	FFprobe            string   `json:"ffprobe"`
	ScanProgram        string   `json:"scan_program"`
	DetailedAnalytics  bool     `json:"detailed_analytics"`
	RetentionDays      int      `json:"retention_days"`
	DriveURL           string   `json:"drive_url"`
	BasePath           string   `json:"-"`
	ConfigPath         string   `json:"-"`
	Setup              bool     `json:"-"`
}

func LoadConfig(p string) (Config, error) {
	c := Config{Listen: "127.0.0.1:8796", PublicURL: "http://127.0.0.1:8796", DataDir: "data", Mode: "managed", SessionHours: 8, UploadMB: 2048, QuotaMB: 10240, StorageMB: 102400, Workers: 1, Threads: 2, JobMinutes: 120, Qualities: []int{360, 480, 720, 1080}, AllowedFormats: []string{".mp4", ".mov", ".mkv", ".webm"}, RetentionDays: 90}
	b, e := os.ReadFile(p)
	if e == nil {
		if e = json.Unmarshal(b, &c); e != nil {
			return c, e
		}
	} else if !os.IsNotExist(e) {
		return c, e
	} else {
		c.Setup = true
	}
	for k, v := range map[string]*string{"MYTUBE_LISTEN": &c.Listen, "MYTUBE_DATA_DIR": &c.DataDir, "MYTUBE_MEDIA_DIR": &c.MediaDir, "MYTUBE_PUBLIC_URL": &c.PublicURL, "WORKSPACE_OIDC_ISSUER": &c.Issuer, "WORKSPACE_OIDC_CLIENT_ID": &c.ClientID, "WORKSPACE_OIDC_CLIENT_SECRET": &c.ClientSecret, "WORKSPACE_OIDC_REDIRECT_URI": &c.RedirectURI, "MYTUBE_OIDC_ISSUER": &c.Issuer, "MYTUBE_OIDC_CLIENT_ID": &c.ClientID, "MYTUBE_OIDC_CLIENT_SECRET": &c.ClientSecret, "MYTUBE_DRIVE_URL": &c.DriveURL} {
		if s := os.Getenv(k); s != "" {
			*v = s
		}
	}
	if !filepath.IsAbs(c.DataDir) {
		c.DataDir = filepath.Join(filepath.Dir(p), c.DataDir)
	}
	if c.MediaDir == "" {
		c.MediaDir = filepath.Join(c.DataDir, "media")
	} else if !filepath.IsAbs(c.MediaDir) {
		c.MediaDir = filepath.Join(filepath.Dir(p), c.MediaDir)
	}
	if c.RedirectURI != "" && os.Getenv("MYTUBE_PUBLIC_URL") == "" {
		c.PublicURL = strings.TrimSuffix(c.RedirectURI, "/auth/callback")
	}
	if c.Issuer != "" {
		c.Setup = false
	}
	if v := os.Getenv("MYTUBE_ADMIN_SUBJECTS"); v != "" {
		c.AdminSubjects = strings.Split(v, ",")
	}
	c.ConfigPath = p
	return c, c.Validate()
}
func loopback(s string) bool {
	ip := net.ParseIP(s)
	return s == "localhost" || ip != nil && ip.IsLoopback()
}
func (c *Config) Validate() error {
	u, e := url.Parse(c.PublicURL)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("public_url must be an absolute HTTP(S) URL")
	}
	if u.Scheme == "http" && !loopback(u.Hostname()) {
		return errors.New("HTTPS is required outside loopback")
	}
	c.PublicURL = strings.TrimRight(c.PublicURL, "/")
	c.BasePath = strings.TrimRight(u.Path, "/")
	host, port, e := net.SplitHostPort(c.Listen)
	if e != nil {
		return e
	}
	n, e := strconv.Atoi(port)
	if e != nil || n < 1024 || n > 65535 {
		return errors.New("port must be 1024–65535")
	}
	if c.Mode != "managed" && c.Mode != "local" {
		return errors.New("mode must be managed or local")
	}
	if c.Mode == "local" || c.Setup {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() || !loopback(u.Hostname()) {
			return errors.New("local/setup mode requires loopback")
		}
	}
	if c.SessionHours < 1 || c.SessionHours > 24 || c.UploadMB < 1 || c.UploadMB > 1048576 || c.QuotaMB < c.UploadMB || c.StorageMB < c.QuotaMB || c.Workers < 1 || c.Workers > 4 || c.Threads < 1 || c.Threads > 8 || c.JobMinutes < 1 || c.JobMinutes > 1440 || c.RetentionDays < 1 || c.RetentionDays > 3650 {
		return errors.New("invalid resource limits")
	}
	for _, q := range c.Qualities {
		if q != 360 && q != 480 && q != 720 && q != 1080 {
			return errors.New("qualities must be 360,480,720,1080")
		}
	}
	if len(c.AllowedFormats) == 0 {
		return errors.New("at least one allowed format is required")
	}
	for _, ext := range c.AllowedFormats {
		if ext != ".mp4" && ext != ".mov" && ext != ".mkv" && ext != ".webm" {
			return errors.New("invalid allowed format")
		}
	}
	for _, hour := range c.ProcessingHoursUTC {
		if hour < 0 || hour > 23 {
			return errors.New("processing hours must be UTC hours 0–23")
		}
	}
	if c.RedirectURI != "" && c.RedirectURI != c.PublicURL+"/auth/callback" {
		return errors.New("OIDC callback does not match public_url")
	}
	for _, s := range []string{c.Issuer, c.DriveURL} {
		if s == "" {
			continue
		}
		v, e := url.Parse(s)
		if e != nil || v.Host == "" || v.User != nil || v.RawQuery != "" || v.Fragment != "" || v.Scheme != "https" && (v.Scheme != "http" || !loopback(v.Hostname())) {
			return errors.New("integration URL requires HTTPS or loopback HTTP")
		}
	}
	return nil
}
func CheckDataDirectory(dir string) error {
	abs, e := filepath.Abs(dir)
	if e != nil {
		return e
	}
	if abs == filepath.VolumeName(abs)+string(filepath.Separator) {
		return errors.New("choose a dedicated My Tube data directory")
	}
	entries, e := os.ReadDir(abs)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	_, db := os.Stat(filepath.Join(abs, "mytube.db"))
	allowed := map[string]bool{"mytube.db": true, "mytube.db-wal": true, "mytube.db-shm": true, "mytube.lock": true, "uploads": true, "backups": true, "media": true, "logs": true}
	for _, ent := range entries {
		if !allowed[ent.Name()] || os.IsNotExist(db) && ent.Name() != "mytube.lock" {
			return errors.New("data directory contains unrelated files")
		}
	}
	return nil
}

func CheckMediaDirectory(dir string) error {
	abs, e := filepath.Abs(dir)
	if e != nil {
		return e
	}
	if abs == filepath.VolumeName(abs)+string(filepath.Separator) {
		return errors.New("choose a dedicated media directory")
	}
	entries, e := os.ReadDir(abs)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	allowed := map[string]bool{"incoming": true, "originals": true, "streaming": true, "thumbnails": true, "subtitles": true}
	for _, entry := range entries {
		if !allowed[entry.Name()] {
			return errors.New("media directory contains unrelated files")
		}
	}
	return nil
}
