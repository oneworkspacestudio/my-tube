package app

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// Standalone upstream helpers are fetched only on the first explicit import.
// Never resolve 'latest', execute a host config/plugin or self-update a helper.
const youtubeToolsVersion = "yt-dlp-2026.10.21-deno-2.9.8"

type youtubeToolPin struct {
	URL, SHA string
	Size     int64
}
type youtubePlatformPins struct{ Download, RuntimeArchive, Runtime youtubeToolPin }

var youtubePins = map[string]youtubePlatformPins{
	"windows": {
		Download:       youtubeToolPin{"https://github.com/yt-dlp/yt-dlp/releases/download/2026.10.21/yt-dlp.exe", "33121d192d706e62f684a06f80918e3f3a8ba52ac3a65213b6951c39c7d9f555", 17880111},
		RuntimeArchive: youtubeToolPin{"https://github.com/denoland/deno/releases/download/v2.9.8/deno-x86_64-pc-windows-msvc.zip", "cba84b26e0df5ae7f74fc66b86a95b866872a86292ebd84fa1e959927edd93bf", 42917472},
		Runtime:        youtubeToolPin{"", "924397800533d3eb0ed0c889b20ad70e18b970d93acd8135bc043f425e528266", 98081568},
	},
	"linux": {
		Download:       youtubeToolPin{"https://github.com/yt-dlp/yt-dlp/releases/download/2026.10.21/yt-dlp_linux", "d8bd33fca9d5b23bd5b9952b3fca36db5c46331933fda9f598a3da14ce2c243f", 40569080},
		RuntimeArchive: youtubeToolPin{"https://github.com/denoland/deno/releases/download/v2.9.8/deno-x86_64-unknown-linux-gnu.zip", "e890975bc2549807bbcc0b584ffafda72cc7c93349dd99c80483dd784f6c4609", 41963064},
		Runtime:        youtubeToolPin{"", "368bca08999e02c6911b8f248167b582276b1f547db5712b6e736d7eda1dbda7", 96055992},
	},
}

func safeYouTubeDir(p string) error {
	if e := os.MkdirAll(p, 0700); e != nil {
		return e
	}
	s, e := os.Lstat(p)
	if e != nil || !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
		return errors.New("YouTube helper folder is not a safe private directory")
	}
	return nil
}
func youtubeToolsDir(data string) string {
	return filepath.Join(data, "helpers", youtubeToolsVersion+"-"+runtime.GOOS)
}
func (a *App) youtubeTools(ctx context.Context, source string) (string, string, error) {
	pins, ok := youtubePins[runtime.GOOS]
	if !ok || runtime.GOARCH != "amd64" {
		return "", "", errors.New("YouTube imports currently support Windows x64 and Linux x64")
	}
	root, e := filepath.Abs(filepath.Join(a.Config.DataDir, "helpers"))
	if e != nil {
		return "", "", e
	}
	if e := safeYouTubeDir(root); e != nil {
		return "", "", e
	}
	dest := filepath.Join(root, youtubeToolsVersion+"-"+runtime.GOOS)
	if e := safeYouTubeDir(dest); e != nil {
		return "", "", e
	}
	tool := filepath.Join(dest, "yt-dlp"+exeExt())
	deno := filepath.Join(dest, "deno"+exeExt())
	if checkedHelper(tool, pins.Download.SHA, pins.Download.Size) && checkedHelper(deno, pins.Runtime.SHA, pins.Runtime.Size) {
		// Linux private-data protection intentionally resets ordinary file modes
		// at startup; restore execution only after both cached hashes are valid.
		if e = os.Chmod(tool, 0700); e != nil {
			return "", "", e
		}
		if e = os.Chmod(deno, 0700); e != nil {
			return "", "", e
		}
		return tool, deno, nil
	}
	// Repair only recognized regular files; preserve unknown files and links.
	entries, e := os.ReadDir(dest)
	if e != nil {
		return "", "", e
	}
	for _, entry := range entries {
		if (entry.Name() != filepath.Base(tool) && entry.Name() != filepath.Base(deno)) || !entry.Type().IsRegular() {
			return "", "", errors.New("Unknown files in YouTube helper cache. Ask the administrator to inspect this folder")
		}
	}
	stage, e := os.MkdirTemp(root, ".youtube-install-")
	if e != nil {
		return "", "", e
	}
	defer func() {
		for _, name := range []string{"yt-dlp" + exeExt(), "deno" + exeExt(), "runtime.zip"} {
			os.Remove(filepath.Join(stage, name))
		}
		os.Remove(stage)
	}()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	if e = a.fetchYouTubeHelper(ctx, source, filepath.Join(stage, filepath.Base(tool)), pins.Download); e != nil {
		return "", "", e
	}
	archive := filepath.Join(stage, "runtime.zip")
	if e = a.fetchYouTubeHelper(ctx, source, archive, pins.RuntimeArchive); e != nil {
		return "", "", e
	}
	z, e := zip.OpenReader(archive)
	if e != nil {
		return "", "", errors.New("YouTube runtime archive is unreadable")
	}
	var found bool
	for _, entry := range z.File {
		if entry.Name != filepath.Base(deno) {
			continue
		}
		if found || entry.UncompressedSize64 != uint64(pins.Runtime.Size) || !entry.Mode().IsRegular() {
			z.Close()
			return "", "", errors.New("Unexpected YouTube runtime archive entry")
		}
		found = true
		in, err := entry.Open()
		if err != nil {
			z.Close()
			return "", "", err
		}
		out, err := os.OpenFile(filepath.Join(stage, filepath.Base(deno)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if err != nil {
			in.Close()
			z.Close()
			return "", "", err
		}
		n, err := io.Copy(out, io.LimitReader(in, pins.Runtime.Size+1))
		in.Close()
		if err == nil {
			err = out.Sync()
		}
		ce := out.Close()
		if err == nil {
			err = ce
		}
		if err != nil || n != pins.Runtime.Size {
			z.Close()
			return "", "", errors.New("YouTube runtime extraction failed")
		}
	}
	z.Close()
	if !found || !checkedHelper(filepath.Join(stage, filepath.Base(deno)), pins.Runtime.SHA, pins.Runtime.Size) {
		return "", "", errors.New("YouTube runtime verification failed")
	}
	for _, name := range []string{filepath.Base(tool), filepath.Base(deno)} {
		p := filepath.Join(dest, name)
		if st, err := os.Lstat(p); err == nil {
			if !st.Mode().IsRegular() {
				return "", "", errors.New("Unsafe YouTube helper cache file")
			}
			if err = os.Remove(p); err != nil {
				return "", "", err
			}
		} else if !os.IsNotExist(err) {
			return "", "", err
		}
		if e = os.Chmod(filepath.Join(stage, name), 0700); e != nil {
			return "", "", e
		}
		if e = os.Rename(filepath.Join(stage, name), p); e != nil {
			return "", "", e
		}
	}
	return tool, deno, nil
}
func (a *App) fetchYouTubeHelper(ctx context.Context, source, dest string, pin youtubeToolPin) error {
	client := &http.Client{Timeout: 15 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		h := r.URL.Hostname()
		if len(via) > 4 || r.URL.Scheme != "https" || r.URL.User != nil || r.URL.Port() != "" || (h != "github.com" && h != "release-assets.githubusercontent.com" && h != "objects.githubusercontent.com") {
			return errors.New("Unexpected YouTube tools download host")
		}
		return nil
	}}
	req, e := http.NewRequestWithContext(ctx, "GET", pin.URL, nil)
	if e != nil {
		return e
	}
	res, e := client.Do(req)
	if e != nil {
		return errors.New("Cannot download verified YouTube tools. Check Internet access and retry")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || (res.ContentLength >= 0 && res.ContentLength != pin.Size) {
		return errors.New("Unexpected YouTube tools download response")
	}
	f, e := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if e != nil {
		return e
	}
	w := &youtubeDownloadWriter{file: f, limit: pin.Size, update: func(n int64) {
		a.youtubeStatus(source, "preparing", "Downloading verified YouTube tools · "+byteProgress(n, pin.Size), false)
	}}
	_, e = io.Copy(w, io.LimitReader(res.Body, pin.Size+1))
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil || !checkedHelper(dest, pin.SHA, pin.Size) {
		return errors.New("YouTube tools download was incomplete or failed verification. Retry to download a fresh copy")
	}
	return nil
}
func byteProgress(n, total int64) string { return fmtYouTubeBytes(n) + " / " + fmtYouTubeBytes(total) }
func fmtYouTubeBytes(n int64) string {
	return strconv.FormatFloat(float64(n)/1048576, 'f', 1, 64) + " MiB"
}
