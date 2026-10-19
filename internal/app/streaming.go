package app

import (
	"bufio"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const streamingProfile = "bounded-hls-v1"

type streamRate struct{ Video, Max, Buffer, Audio int }

func rateForHeight(height int) streamRate {
	switch {
	case height <= 360:
		return streamRate{450, 650, 1300, 96}
	case height <= 480:
		return streamRate{800, 1100, 2200, 96}
	case height <= 720:
		return streamRate{1500, 2000, 4000, 128}
	default:
		return streamRate{3000, 4000, 8000, 128}
	}
}

// Existing completed originals are queued once for the new bounded profile.
// No media is removed or overwritten until a new complete rendition is ready.
func (a *App) queueUnoptimised() {
	rows, e := a.Store.DB.Query("SELECT v.id FROM videos v JOIN jobs j ON j.video_id=v.id WHERE v.deleted=0 AND j.status='ready'")
	if e != nil {
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && objectID.MatchString(id) {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		p, e := os.ReadFile(filepath.Join(a.Config.MediaDir, "streaming", id, "profile.txt"))
		master, masterErr := os.Stat(filepath.Join(a.Config.MediaDir, "streaming", id, "master.m3u8"))
		if e == nil && string(p) == streamingProfile && masterErr == nil && master.Mode().IsRegular() {
			continue
		}
		a.Store.DB.Exec("UPDATE jobs SET status='queued',error='',updated=? WHERE video_id=? AND status='ready'", now(), id)
	}
}

// Measure the multiplexed output including audio/TS overhead instead of guessing
// BANDWIDTH from height. Conservative peak advertises the largest real segment.
func variantBandwidth(file string) (int64, int64, error) {
	f, e := os.Open(file)
	if e != nil {
		return 0, 0, e
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var seconds, totalSeconds, peak float64
	var total int64
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#EXTINF:") {
			seconds, e = strconv.ParseFloat(strings.Split(strings.TrimPrefix(line, "#EXTINF:"), ",")[0], 64)
			if e != nil || seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
				return 0, 0, errors.New("Invalid segment duration")
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if seconds <= 0 || !mediaName.MatchString(line) || !strings.HasSuffix(line, ".ts") {
			return 0, 0, errors.New("Invalid local segment")
		}
		info, e := os.Stat(filepath.Join(filepath.Dir(file), line))
		if e != nil || !info.Mode().IsRegular() {
			return 0, 0, errors.New("Missing streaming segment")
		}
		total += info.Size()
		totalSeconds += seconds
		peak = math.Max(peak, float64(info.Size())*8/seconds)
		seconds = 0
		count++
	}
	if e = scanner.Err(); e != nil {
		return 0, 0, e
	}
	if count == 0 || totalSeconds <= 0 {
		return 0, 0, errors.New("No streaming segments")
	}
	return int64(math.Ceil(peak * 1.15)), int64(math.Ceil(float64(total) * 8 / totalSeconds)), nil
}
