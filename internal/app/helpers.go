package app

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type helperProgress struct {
	Phase      string `json:"phase"`
	Message    string `json:"message"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
}
type mediaHelpers struct {
	install    sync.Mutex
	mu         sync.Mutex
	progress   helperProgress
	dir        string
	attempted  time.Time
	lastError  error
	retryAfter time.Time
}

var errMediaToolsPending = errors.New("Waiting for verified video tools; original remains playable")

func (a *App) helperRetryBlocked() bool {
	a.helpers.mu.Lock()
	defer a.helpers.mu.Unlock()
	return time.Now().Before(a.helpers.retryAfter)
}

func (a *App) setHelperProgress(p helperProgress) {
	a.helpers.mu.Lock()
	a.helpers.progress = p
	a.helpers.mu.Unlock()
}
func (a *App) helperProgress() helperProgress {
	a.helpers.mu.Lock()
	defer a.helpers.mu.Unlock()
	p := a.helpers.progress
	if p.Phase == "" {
		p.Phase = "idle"
		p.Message = "Adaptive quality prepares automatically when a video is uploaded."
		if !a.Config.Transcode {
			p.Message = "Video optimisation is disabled; playback uses the original file."
		}
	}
	return p
}
func (a *App) downloadedHelper(name string) string {
	a.helpers.mu.Lock()
	defer a.helpers.mu.Unlock()
	if a.helpers.dir == "" {
		return ""
	}
	return filepath.Join(a.helpers.dir, name+".exe")
}
func checkedHelper(file, want string, size int64) bool {
	info, e := os.Lstat(file)
	if e != nil || !info.Mode().IsRegular() || info.Size() != size {
		return false
	}
	f, e := os.Open(file)
	if e != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return e == nil && hex.EncodeToString(h.Sum(nil)) == want
}

// Helpers are separate upstream programs. The app downloads an immutable,
// checksum-pinned archive directly from its publisher; it ships no FFmpeg code.
func (a *App) prepareMedia(ctx context.Context) error {
	if a.helper(a.Config.FFmpeg, "ffmpeg") != "" && (a.Config.FFmpeg != "" || !a.Config.AutoHelpers || a.helper(a.Config.FFprobe, "ffprobe") != "") {
		return nil
	}
	if !a.Config.AutoHelpers {
		return errors.New("Video optimisation needs FFmpeg and FFprobe. Enable automatic media tools in server settings or configure their paths.")
	}
	if a.Config.FFmpeg != "" || a.Config.FFprobe != "" {
		return errors.New("A configured media tool is unavailable. Correct its path or clear the custom paths to use automatic media tools.")
	}
	a.helpers.install.Lock()
	defer a.helpers.install.Unlock()
	if a.downloadedHelper("ffmpeg") != "" {
		return nil
	}
	if a.helpers.lastError != nil && time.Since(a.helpers.attempted) < time.Minute {
		return a.helpers.lastError
	}
	a.helpers.attempted = time.Now()
	e := a.installMediaTools(ctx)
	a.helpers.lastError = e
	if e != nil {
		a.helpers.mu.Lock()
		a.helpers.retryAfter = time.Now().Add(time.Minute)
		a.helpers.mu.Unlock()
		a.setHelperProgress(helperProgress{Phase: "error", Message: "Video tools could not be prepared. Originals are safe. Check Internet access and retry processing."})
	}
	return e
}
func (a *App) installMediaTools(parent context.Context) error {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return errors.New("Automatic video tools currently require Windows x64; configure FFmpeg and FFprobe paths on this platform")
	}
	root := filepath.Join(a.Config.DataDir, "helpers")
	if e := os.MkdirAll(root, 0700); e != nil {
		return e
	}
	if info, e := os.Lstat(root); e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Unsafe media tools folder")
	}
	dest := filepath.Join(root, "ffmpeg-"+mediaToolsVersion)
	if info, e := os.Lstat(dest); e == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return errors.New("Unsafe versioned media tools folder")
	}
	valid := true
	for name, pin := range mediaToolFiles {
		if !checkedHelper(filepath.Join(dest, name), pin.SHA256, pin.Size) {
			valid = false
		}
	}
	if valid {
		a.helpers.mu.Lock()
		a.helpers.dir = dest
		a.helpers.progress = helperProgress{Phase: "ready", Message: "Adaptive video tools are ready."}
		a.helpers.mu.Unlock()
		return nil
	}
	if _, e := os.Lstat(dest); e == nil {
		entries, e := os.ReadDir(dest)
		if e != nil {
			return e
		}
		for _, entry := range entries {
			if _, ok := mediaToolFiles[entry.Name()]; !ok || entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
				return errors.New("Unknown files in the video tools cache; repair it in advanced settings")
			}
		}
		for _, entry := range entries {
			if e = os.Remove(filepath.Join(dest, entry.Name())); e != nil {
				return e
			}
		}
		if e = os.Remove(dest); e != nil {
			return e
		}
	}
	stage, e := os.MkdirTemp(root, ".install-")
	if e != nil {
		return e
	}
	// Delete only files created here; never recursively remove unknown helper state.
	defer func() {
		for name := range mediaToolFiles {
			os.Remove(filepath.Join(stage, name))
		}
		os.Remove(filepath.Join(stage, "download.zip"))
		os.Remove(stage)
	}()
	ctx, cancel := context.WithTimeout(parent, 12*time.Minute)
	defer cancel()
	client := &http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error {
		h := r.URL.Hostname()
		if len(via) > 4 || r.URL.Scheme != "https" || (h != "github.com" && h != "release-assets.githubusercontent.com" && h != "objects.githubusercontent.com") {
			return errors.New("Unexpected media tools download host")
		}
		return nil
	}}
	req, e := http.NewRequestWithContext(ctx, "GET", mediaToolsURL, nil)
	if e != nil {
		return e
	}
	req.Header.Set("User-Agent", "OneWorkspace-MyTube/"+Version)
	a.setHelperProgress(helperProgress{Phase: "downloading", Message: "Downloading verified video tools (one time)…", Total: mediaToolsBytes})
	response, e := client.Do(req)
	if e != nil {
		return errors.New("Video tools download failed")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || (response.ContentLength > 0 && response.ContentLength != mediaToolsBytes) {
		return errors.New("Unexpected video tools download response")
	}
	archive := filepath.Join(stage, "download.zip")
	f, e := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	hash := sha256.New()
	reader := io.LimitReader(response.Body, mediaToolsBytes+1)
	buffer := make([]byte, 256*1024)
	var total int64
	for {
		n, readErr := reader.Read(buffer)
		if n > 0 {
			if _, e = io.Copy(io.MultiWriter(f, hash), bytes.NewReader(buffer[:n])); e != nil {
				f.Close()
				return e
			}
			total += int64(n)
			a.setHelperProgress(helperProgress{Phase: "downloading", Message: "Downloading verified video tools (one time)…", Total: mediaToolsBytes, Downloaded: total})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			f.Close()
			return errors.New("Video tools download was interrupted")
		}
	}
	if e = f.Close(); e != nil {
		return e
	}
	if total != mediaToolsBytes || hex.EncodeToString(hash.Sum(nil)) != mediaToolsSHA256 {
		return errors.New("Video tools checksum did not match; nothing was executed")
	}
	a.setHelperProgress(helperProgress{Phase: "verifying", Message: "Checking and preparing video tools…", Downloaded: total, Total: total})
	z, e := zip.OpenReader(archive)
	if e != nil {
		return e
	}
	found := map[string]bool{}
	for _, member := range z.File {
		for name, pin := range mediaToolFiles {
			if member.Name != pin.Member {
				continue
			}
			if found[name] || member.UncompressedSize64 != uint64(pin.Size) {
				z.Close()
				return errors.New("Unexpected media tool archive member")
			}
			found[name] = true
			source, e := member.Open()
			if e != nil {
				z.Close()
				return e
			}
			target := filepath.Join(stage, name)
			output, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
			if e != nil {
				source.Close()
				z.Close()
				return e
			}
			_, copyErr := io.Copy(output, io.LimitReader(source, pin.Size+1))
			source.Close()
			closeErr := output.Close()
			if copyErr != nil || closeErr != nil || !checkedHelper(target, pin.SHA256, pin.Size) {
				z.Close()
				return fmt.Errorf("Media tool verification failed: %s", name)
			}
		}
	}
	z.Close()
	if len(found) != len(mediaToolFiles) {
		return errors.New("Media tool archive is incomplete")
	}
	os.Remove(archive)
	if e = os.Rename(stage, dest); e != nil {
		return e
	}
	a.helpers.mu.Lock()
	a.helpers.dir = dest
	a.helpers.progress = helperProgress{Phase: "ready", Message: "Adaptive video tools are ready."}
	a.helpers.mu.Unlock()
	return nil
}
