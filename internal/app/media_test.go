package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProcessingFailurePreservesOriginalAndRestartsJobs(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	channel(t, a, s)
	id := uploadVideo(t, a, s, "private")
	a.Config.FFmpeg = filepath.Join(t.TempDir(), "missing.exe")
	a.Config.Transcode = true
	v, e := a.video(id, s.User.ID)
	if e != nil {
		t.Fatal(e)
	}
	v.Direct = false
	a.Store.DB.Exec("UPDATE videos SET direct=0 WHERE id=?", id)
	if e = a.process(context.Background(), id); e == nil {
		t.Fatal("Missing helper should report failure")
	}
	if _, e = os.Stat(a.original(v)); e != nil {
		t.Fatal("Lost original")
	}
	a.Store.DB.Exec("UPDATE jobs SET status='running' WHERE video_id=?", id)
	if e = a.Recover(); e != nil {
		t.Fatal(e)
	}
	var status string
	a.Store.DB.QueryRow("SELECT status FROM jobs WHERE video_id=?", id).Scan(&status)
	if status != "queued" {
		t.Fatal(status)
	}
}
func TestRealFFmpegHLSAndPoster(t *testing.T) {
	ff := os.Getenv("MYTUBE_TEST_FFMPEG")
	probe := os.Getenv("MYTUBE_TEST_FFPROBE")
	if ff == "" || probe == "" {
		t.Skip("Optional real helper integration: set MYTUBE_TEST_FFMPEG and MYTUBE_TEST_FFPROBE")
	}
	a := testApp(t)
	a.Config.FFmpeg = ff
	a.Config.FFprobe = probe
	a.Config.Transcode = true
	s := login(t, a, "alice")
	other := login(t, a, "bob")
	channel(t, a, s)
	id := uploadVideo(t, a, s, "private")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if e := a.process(ctx, id); e != nil {
		t.Fatal(e)
	}
	master, e := os.ReadFile(filepath.Join(a.Config.MediaDir, "streaming", id, "master.m3u8"))
	if e != nil || !strings.Contains(string(master), "q360.m3u8") || strings.Contains(string(master), "q480") {
		t.Fatalf("Tiers: %s %v", master, e)
	}
	for _, file := range []string{"master.m3u8", "q360.m3u8", "q360-00000.ts", "poster.jpg"} {
		mustStatus(t, call(t, a, s, "GET", "/api/videos/"+id+"/media/"+file, nil, nil), 200)
		mustStatus(t, call(t, a, other, "GET", "/api/videos/"+id+"/media/"+file, nil, nil), 404)
	}
}
