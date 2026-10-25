package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"math"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Imported channels are owner-only source collections. Keeping them separate
// from creator channels preserves existing identity, uploads and foreign keys.
const youtubeChannelCols = "s.id,s.owner,c.name,c.handle,c.description,s.url,s.playlist_id,s.state,s.message"

func scanYouTubeChannel(r scanner) (Channel, error) {
	c := Channel{Imported: true, Visibility: "private"}
	e := r.Scan(&c.ID, &c.Owner, &c.Name, &c.Handle, &c.Description, &c.SourceURL, &c.Playlist, &c.ImportState, &c.ImportMessage)
	return c, e
}

func (a *App) youtubeChannels(uid string) ([]Channel, error) {
	out := []Channel{}
	rows, e := a.Store.DB.Query("SELECT "+youtubeChannelCols+" FROM youtube_channels c JOIN youtube_sources s ON s.id=c.source_id WHERE s.owner=? ORDER BY c.name", uid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		c, e := scanYouTubeChannel(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (a *App) youtubeChannel(id, uid string) (Channel, error) {
	if uid == "" || !objectID.MatchString(id) {
		return Channel{}, sql.ErrNoRows
	}
	c, e := scanYouTubeChannel(a.Store.DB.QueryRow("SELECT "+youtubeChannelCols+" FROM youtube_channels c JOIN youtube_sources s ON s.id=c.source_id WHERE s.id=? AND s.owner=?", id, uid))
	if e != nil {
		return c, e
	}
	rows, e := a.Store.DB.Query("SELECT remote_id,title,state,message,video_id,bytes,total FROM youtube_items WHERE source_id=? ORDER BY updated DESC,remote_id LIMIT 100", id)
	if e != nil {
		return c, e
	}
	defer rows.Close()
	c.Items = []youtubeItem{}
	for rows.Next() {
		var v youtubeItem
		if e = rows.Scan(&v.Remote, &v.Title, &v.State, &v.Message, &v.Video, &v.Bytes, &v.Total); e != nil {
			return c, e
		}
		c.Items = append(c.Items, v)
	}
	return c, rows.Err()
}

func youtubeExitCode(e error) string {
	if exit, ok := e.(*exec.ExitError); ok {
		return "exit " + strconv.Itoa(exit.ExitCode())
	}
	return "could not start video tool"
}

// FFmpeg can otherwise exit successfully after truncated network input. Check
// each stream, not only container duration: audio may end before the video.
func (a *App) verifyYouTubeDownload(ctx context.Context, path string, expected float64) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out := &boundedYouTubeBuffer{limit: 1 << 20}
	args := []string{"-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov", "-show_entries", "stream=codec_type,duration", "-of", "json", path}
	if e := a.run(ctx, a.helper(a.Config.FFprobe, "ffprobe"), args, out); e != nil {
		return errors.New("The downloaded video could not be verified; retry this import")
	}
	var result struct {
		Streams []struct {
			Kind     string `json:"codec_type"`
			Duration string `json:"duration"`
		} `json:"streams"`
	}
	if json.Unmarshal(out.Bytes(), &result) != nil {
		return errors.New("The downloaded video returned unreadable verification data")
	}
	video, audio := false, false
	for _, s := range result.Streams {
		if s.Kind != "video" && s.Kind != "audio" {
			continue
		}
		d, e := strconv.ParseFloat(s.Duration, 64)
		if e != nil || math.IsNaN(d) || math.IsInf(d, 0) || d <= 0 || expected > 0 && d < expected-math.Max(3, expected*.005) {
			return errors.New("YouTube closed the download before all audio and video arrived. Retry this import; no incomplete video was saved")
		}
		video = video || s.Kind == "video"
		audio = audio || s.Kind == "audio"
	}
	if !video || !audio {
		return errors.New("YouTube did not provide complete audio and video; retry this import")
	}
	return nil
}

// Metadata cannot turn artwork fetching into arbitrary network access. Both
// the initial URL and every redirect are restricted to YouTube image hosts.
func youtubeArtworkURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	return u.Hostname() == "yt3.ggpht.com" || u.Hostname() == "yt3.googleusercontent.com"
}
func (a *App) youtubeArtwork(ctx context.Context, id string, listing youtubeEntry) {
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 3 || !youtubeArtworkURL(r.URL.String()) {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	// Artwork is optional; a failed image never prevents importing the channel.
	for _, kind := range []string{"avatar", "banner"} {
		raw := ""
		for _, t := range listing.Thumbnails {
			if strings.Contains(t.ID, kind) && youtubeArtworkURL(t.URL) {
				raw = t.URL
			}
		}
		if raw == "" {
			continue
		}
		req, e := http.NewRequestWithContext(ctx, "GET", raw, nil)
		if e != nil {
			continue
		}
		res, e := client.Do(req)
		if e != nil {
			continue
		}
		b, e := io.ReadAll(io.LimitReader(res.Body, (5<<20)+1))
		res.Body.Close()
		if e != nil || res.StatusCode != 200 || len(b) > 5<<20 {
			continue
		}
		cfg, format, e := image.DecodeConfig(bytes.NewReader(b))
		if e != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || cfg.Width*cfg.Height > 24000000 {
			continue
		}
		img, _, e := image.Decode(bytes.NewReader(b))
		if e != nil {
			continue
		}
		var encoded bytes.Buffer
		if jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 88}) != nil {
			continue
		}
		atomicWrite(filepath.Join(a.Config.MediaDir, "thumbnails", id+"-"+kind+".jpg"), encoded.Bytes())
	}
}
