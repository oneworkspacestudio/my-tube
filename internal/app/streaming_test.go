package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMeasuredBandwidthAndBoundedLadder(t *testing.T) {
	for _, h := range []int{240, 360, 480, 720, 1080} {
		r := rateForHeight(h)
		if r.Video <= 0 || r.Max < r.Video || r.Buffer < r.Max || r.Max > 4000 {
			t.Fatal(h, r)
		}
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "q360-00000.ts"), make([]byte, 100000), 0600)
	os.WriteFile(filepath.Join(dir, "q360-00001.ts"), make([]byte, 200000), 0600)
	p := filepath.Join(dir, "q360.m3u8")
	os.WriteFile(p, []byte("#EXTM3U\n#EXTINF:4.000,\nq360-00000.ts\n#EXTINF:4.000,\nq360-00001.ts\n#EXT-X-ENDLIST\n"), 0600)
	peak, average, e := variantBandwidth(p)
	if e != nil || peak < 460000 || peak > 460001 || average != 300000 {
		t.Fatal(peak, average, e)
	}
	os.WriteFile(p, []byte("#EXTINF:4,\n../outside.ts\n"), 0600)
	if _, _, e = variantBandwidth(p); e == nil {
		t.Fatal("accepted unsafe segment")
	}
	os.WriteFile(p, []byte("#EXTINF:NaN,\nq360-00000.ts\n"), 0600)
	if _, _, e = variantBandwidth(p); e == nil {
		t.Fatal("accepted invalid duration")
	}
}
func TestExistingOriginalQueuesForOptimisationOnce(t *testing.T) {
	a := testApp(t)
	s := login(t, a, "alice")
	channel(t, a, s)
	id := uploadVideo(t, a, s, "private")
	a.Store.DB.Exec("UPDATE jobs SET status='ready' WHERE video_id=?", id)
	a.queueUnoptimised()
	var state string
	a.Store.DB.QueryRow("SELECT status FROM jobs WHERE video_id=?", id).Scan(&state)
	if state != "queued" {
		t.Fatal(state)
	}
	folder := filepath.Join(a.Config.MediaDir, "streaming", id)
	os.MkdirAll(folder, 0700)
	os.WriteFile(filepath.Join(folder, "profile.txt"), []byte(streamingProfile), 0600)
	os.WriteFile(filepath.Join(folder, "master.m3u8"), []byte("#EXTM3U"), 0600)
	a.Store.DB.Exec("UPDATE jobs SET status='ready' WHERE video_id=?", id)
	a.queueUnoptimised()
	a.Store.DB.QueryRow("SELECT status FROM jobs WHERE video_id=?", id).Scan(&state)
	if state != "ready" {
		t.Fatal(state)
	}
	v, e := a.video(id, s.User.ID)
	if e != nil {
		t.Fatal(e)
	}
	original, e := os.ReadFile(a.original(v))
	source, _ := os.ReadFile("testdata/sample.mp4")
	if e != nil || !bytes.Equal(original, source) {
		t.Fatal("original changed", e)
	}
}
