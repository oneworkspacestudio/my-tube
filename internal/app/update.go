package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// Explicit, administrator-only check. No credentials or automatic replacement.
func (a *App) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/oneworkspacestudio/my-tube/releases?per_page=5", nil)
	if e != nil {
		problem(w, 500, "Cannot prepare update check")
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MyTube/"+Version)
	client := http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := client.Do(req)
	if e != nil {
		problem(w, 503, "GitHub release check is unavailable")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		problem(w, 503, "GitHub release check is rate-limited or unavailable")
		return
	}
	var releases []struct {
		Tag        string `json:"tag_name"`
		URL        string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if e = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&releases); e != nil {
		problem(w, 503, "Invalid release response")
		return
	}
	for _, v := range releases {
		if !v.Draft && strings.HasPrefix(v.URL, "https://github.com/oneworkspacestudio/my-tube/releases/tag/") {
			respond(w, 200, map[string]any{"current": Version, "latest": strings.TrimPrefix(v.Tag, "v"), "url": v.URL, "prerelease": v.Prerelease})
			return
		}
	}
	respond(w, 200, map[string]any{"current": Version, "latest": "", "url": "https://github.com/oneworkspacestudio/my-tube/releases"})
}
