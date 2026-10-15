package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Metadata struct {
	Duration      float64
	Width, Height int
	Direct        bool
}

func (a *App) helper(config, name string) string {
	if config != "" {
		if p, e := exec.LookPath(config); e == nil {
			return p
		}
		return ""
	}
	exe, _ := os.Executable()
	p := filepath.Join(filepath.Dir(exe), "helpers", name+exeExt())
	if _, e := os.Stat(p); e == nil {
		return p
	}
	return ""
}
func (a *App) inspect(ctx context.Context, p string) (Metadata, error) {
	f, e := os.Open(p)
	if e != nil {
		return Metadata{}, e
	}
	head := make([]byte, 16)
	n, _ := f.Read(head)
	f.Close()
	if n < 12 {
		return Metadata{}, errors.New("File is not a supported video container")
	}
	mp4 := string(head[4:8]) == "ftyp" || string(head[4:8]) == "wide" || string(head[4:8]) == "mdat"
	ebml := bytes.Equal(head[:4], []byte{0x1a, 0x45, 0xdf, 0xa3})
	if !mp4 && !ebml {
		return Metadata{}, errors.New("Container signature is not MP4/MOV or Matroska/WebM")
	}
	probe := a.helper(a.Config.FFprobe, "ffprobe")
	if probe == "" {
		if !mp4 {
			return Metadata{}, errors.New("Configure FFprobe for MKV/WebM/MOV codec analysis")
		}
		return inspectMP4(p)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var output bytes.Buffer
	args := []string{"-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-show_streams", "-show_format", "-of", "json", p}
	if e = a.run(ctx, probe, args, &output); e != nil {
		return Metadata{}, errors.New("FFprobe could not decode this video")
	}
	var data struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			PixFmt    string `json:"pix_fmt"`
		}
		Format struct {
			Duration   string `json:"duration"`
			FormatName string `json:"format_name"`
		}
	}
	if e = json.Unmarshal(output.Bytes(), &data); e != nil {
		return Metadata{}, e
	}
	m := Metadata{}
	m.Duration, _ = strconv.ParseFloat(data.Format.Duration, 64)
	compatible := mp4
	videoCount := 0
	for _, s := range data.Streams {
		if s.CodecType == "video" {
			videoCount++
			if videoCount == 1 {
				m.Width = s.Width
				m.Height = s.Height
			}
			if s.CodecName != "h264" || s.PixFmt != "yuv420p" {
				compatible = false
			}
		}
		if s.CodecType == "audio" && s.CodecName != "aac" {
			compatible = false
		}
	}
	m.Direct = compatible && videoCount == 1
	if m.Duration <= 0 || m.Duration > 86400 || m.Width < 1 || m.Height < 1 || m.Width > 16384 || m.Height > 16384 {
		return m, errors.New("Unsupported duration or dimensions")
	}
	return m, nil
}

// Conservative bounded ISO-BMFF parser keeps verified H.264/AAC MP4 usable without helpers.
// This path rejects fragmented/encrypted/unknown codecs rather than guessing from an extension.
func inspectMP4(p string) (Metadata, error) {
	f, e := os.Open(p)
	if e != nil {
		return Metadata{}, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return Metadata{}, e
	}
	m := Metadata{}
	var moov []byte
	mdat := false
	pos := int64(0)
	for count := 0; pos < st.Size() && count < 100000; count++ {
		var h [16]byte
		if _, e = f.ReadAt(h[:8], pos); e != nil {
			return m, errors.New("Truncated MP4 box")
		}
		size := int64(binary.BigEndian.Uint32(h[:4]))
		header := int64(8)
		if size == 1 {
			if _, e = f.ReadAt(h[8:], pos+8); e != nil {
				return m, e
			}
			size = int64(binary.BigEndian.Uint64(h[8:]))
			header = 16
		}
		if size == 0 {
			size = st.Size() - pos
		}
		if size < header || size > st.Size()-pos {
			return m, errors.New("Invalid MP4 box length")
		}
		switch string(h[4:8]) {
		case "mdat":
			mdat = true
		case "moov":
			if size > 32<<20 {
				return m, errors.New("MP4 metadata too large")
			}
			moov = make([]byte, size-header)
			if _, e = f.ReadAt(moov, pos+header); e != nil {
				return m, e
			}
		case "moof":
			return m, errors.New("Fragmented MP4 requires FFprobe")
		}
		pos += size
	}
	if !mdat || len(moov) == 0 || pos != st.Size() {
		return m, errors.New("MP4 requires complete moov and mdat boxes")
	}
	codecs := 0
	compatible := true
	var walk func([]byte, int) error
	walk = func(b []byte, depth int) error {
		if depth > 12 {
			return errors.New("MP4 nesting too deep")
		}
		for len(b) > 0 {
			if len(b) < 8 {
				return errors.New("Incomplete nested box")
			}
			n := int(binary.BigEndian.Uint32(b))
			if n < 8 || n > len(b) {
				return errors.New("Invalid nested box")
			}
			kind := string(b[4:8])
			v := b[8:n]
			switch kind {
			case "trak", "mdia", "minf", "stbl":
				if e := walk(v, depth+1); e != nil {
					return e
				}
			case "mvhd":
				if len(v) < 20 {
					return errors.New("Invalid movie header")
				}
				if v[0] == 0 {
					scale := binary.BigEndian.Uint32(v[12:16])
					if scale > 0 {
						m.Duration = float64(binary.BigEndian.Uint32(v[16:20])) / float64(scale)
					}
				} else if v[0] == 1 && len(v) >= 32 {
					scale := binary.BigEndian.Uint32(v[20:24])
					if scale > 0 {
						m.Duration = float64(binary.BigEndian.Uint64(v[24:32])) / float64(scale)
					}
				}
			case "stsd":
				if len(v) < 8 {
					return errors.New("Invalid sample descriptions")
				}
				entries := v[8:]
				for i := uint32(0); i < binary.BigEndian.Uint32(v[4:8]); i++ {
					if len(entries) < 8 {
						return errors.New("Invalid sample entry")
					}
					size := int(binary.BigEndian.Uint32(entries))
					if size < 8 || size > len(entries) {
						return errors.New("Invalid sample size")
					}
					codec := string(entries[4:8])
					switch codec {
					case "avc1":
						if size < 86 {
							return errors.New("Invalid AVC sample")
						}
						codecs++
						avcc := false
						for extra := entries[86:size]; len(extra) >= 8; {
							n := int(binary.BigEndian.Uint32(extra))
							if n < 8 || n > len(extra) {
								return errors.New("Invalid AVC configuration")
							}
							if string(extra[4:8]) == "avcC" {
								if n < 15 || extra[8] != 1 {
									return errors.New("Invalid AVC decoder configuration")
								}
								profile := extra[9]
								if profile != 66 && profile != 77 && profile != 88 && profile != 100 {
									return errors.New("This AVC profile requires FFprobe")
								}
								avcc = true
							}
							extra = extra[n:]
						}
						if !avcc {
							return errors.New("Missing AVC decoder configuration")
						}
						m.Width = int(binary.BigEndian.Uint16(entries[32:34]))
						m.Height = int(binary.BigEndian.Uint16(entries[34:36]))
					case "mp4a":
					case "text", "tx3g", "wvtt":
					default:
						compatible = false
					}
					entries = entries[size:]
				}
			}
			b = b[n:]
		}
		return nil
	}
	if e = walk(moov, 0); e != nil {
		return m, e
	}
	m.Direct = compatible && codecs == 1
	if !m.Direct || m.Duration <= 0 || m.Duration > 86400 || m.Width <= 0 || m.Height <= 0 {
		return m, errors.New("Configure FFprobe and FFmpeg for this codec or MP4 layout")
	}
	return m, nil
}

type boundedWriter struct {
	w io.Writer
	n int64
}

func (b *boundedWriter) Write(p []byte) (int, error) {
	if b.n+int64(len(p)) > 4<<20 {
		return 0, errors.New("Helper output exceeded limit")
	}
	b.n += int64(len(p))
	if b.w != nil {
		return b.w.Write(p)
	}
	return len(p), nil
}
func (a *App) run(ctx context.Context, program string, args []string, out io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				used, e := treeSize(a.Config.MediaDir)
				if e != nil || used > a.Config.StorageMB<<20 {
					cancel()
					return
				}
			}
		}
	}()
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdout = &boundedWriter{w: out}
	cmd.Stderr = &boundedWriter{}
	cmd.WaitDelay = 3 * time.Second
	return runLimited(cmd)
}
func (a *App) Media(w http.ResponseWriter, r *http.Request) {
	v, e := a.video(r.PathValue("id"), who(r).User.ID)
	if e != nil {
		problem(w, 404, "Video unavailable")
		return
	}
	name := r.PathValue("file")
	var p string
	switch name {
	case "original":
		if !v.Direct && r.URL.Query().Get("download") != "1" {
			problem(w, 415, "This codec requires a streaming rendition")
			return
		}
		p = filepath.Join(a.Config.MediaDir, "originals", v.Original)
		if r.URL.Query().Get("download") == "1" {
			w.Header().Set("Content-Disposition", `attachment; filename="video`+filepath.Ext(v.Original)+`"`)
		} else {
			w.Header().Set("Content-Type", "video/mp4")
		}
	case "poster.jpg":
		p = filepath.Join(a.Config.MediaDir, "thumbnails", v.ID+".jpg")
		w.Header().Set("Content-Type", "image/jpeg")
	case "captions.vtt":
		p = filepath.Join(a.Config.MediaDir, "subtitles", v.ID+".vtt")
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	default:
		if !mediaName.MatchString(name) {
			problem(w, 404, "Media unavailable")
			return
		}
		p = filepath.Join(a.Config.MediaDir, "streaming", v.ID, name)
		if strings.HasSuffix(name, ".m3u8") {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		} else {
			w.Header().Set("Content-Type", "video/mp2t")
		}
	}
	f, e := os.Open(p)
	if e != nil {
		problem(w, 404, "Media is missing or still processing")
		return
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		problem(w, 404, "Media unavailable")
		return
	}
	http.ServeContent(w, r, name, st.ModTime(), f)
}

var mediaName = regexp.MustCompile(`^(master\.m3u8|q(360|480|720|1080|[1-9][0-9]{1,2})\.m3u8|q[0-9]{2,4}-[0-9]{5,8}\.ts)$`)

func (a *App) Thumbnail(w http.ResponseWriter, r *http.Request) {
	v, e := a.video(r.PathValue("id"), who(r).User.ID)
	if e != nil || v.Owner != who(r).User.ID {
		problem(w, 404, "Video unavailable")
		return
	}
	a.saveImage(w, r, filepath.Join(a.Config.MediaDir, "thumbnails", v.ID+".jpg"))
}
func (a *App) saveImage(w http.ResponseWriter, r *http.Request, dest string) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	b, e := io.ReadAll(r.Body)
	if e != nil {
		problem(w, 400, "Image limit is 5 MiB")
		return
	}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(b))
	if e != nil || (format != "jpeg" && format != "png") || cfg.Width > 8192 || cfg.Height > 8192 || cfg.Width < 1 || cfg.Height < 1 {
		problem(w, 400, "Use a JPEG or PNG up to 8192 pixels")
		return
	}
	if cfg.Width*cfg.Height > 24000000 {
		problem(w, 400, "Image exceeds 24 megapixels")
		return
	}
	img, _, e := image.Decode(bytes.NewReader(b))
	if e != nil {
		problem(w, 400, "Cannot decode image")
		return
	}
	var encoded bytes.Buffer
	if e = jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 88}); e != nil {
		problem(w, 400, "Cannot encode image")
		return
	}
	b = encoded.Bytes()
	if e = atomicWrite(dest, b); e != nil {
		problem(w, 500, "Could not save image")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}
func atomicWrite(dest string, b []byte) error {
	p := dest + ".tmp-" + ID()
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer os.Remove(p)
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(p, dest)
}
func (a *App) Subtitles(w http.ResponseWriter, r *http.Request) {
	v, e := a.video(r.PathValue("id"), who(r).User.ID)
	if e != nil || v.Owner != who(r).User.ID {
		problem(w, 404, "Video unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	b, e := io.ReadAll(r.Body)
	if e != nil || !bytes.HasPrefix(b, []byte("WEBVTT")) || bytes.Contains(b, []byte("<script")) {
		problem(w, 400, "Use a WebVTT subtitle file under 1 MiB")
		return
	}
	e = atomicWrite(filepath.Join(a.Config.MediaDir, "subtitles", v.ID+".vtt"), b)
	a.failure(w, e)
}
func (a *App) ChannelImage(w http.ResponseWriter, r *http.Request) {
	id, e := a.ownerChannel(who(r).User.ID)
	kind := r.PathValue("kind")
	if e != nil || kind != "avatar" && kind != "banner" {
		problem(w, 404, "Channel unavailable")
		return
	}
	a.saveImage(w, r, filepath.Join(a.Config.MediaDir, "thumbnails", id+"-"+kind+".jpg"))
}
func (a *App) GetChannelImage(w http.ResponseWriter, r *http.Request) {
	if !objectID.MatchString(r.PathValue("id")) {
		problem(w, 404, "Channel unavailable")
		return
	}
	uid := who(r).User.ID
	var owner, vis string
	e := a.Store.DB.QueryRow("SELECT owner,visibility FROM channels WHERE id=?", r.PathValue("id")).Scan(&owner, &vis)
	kind := r.PathValue("kind")
	if e != nil || kind != "avatar" && kind != "banner" || owner != uid && (vis == "private" || uid == "" && !a.Config.PublicAccess) {
		problem(w, 404, "Channel unavailable")
		return
	}
	http.ServeFile(w, r, filepath.Join(a.Config.MediaDir, "thumbnails", r.PathValue("id")+"-"+kind+".jpg"))
}
func (a *App) worker(ctx context.Context) {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		var id, vid string
		if len(a.Config.ProcessingHoursUTC) > 0 {
			allowed := false
			for _, h := range a.Config.ProcessingHoursUTC {
				if h == time.Now().UTC().Hour() {
					allowed = true
				}
			}
			if !allowed {
				continue
			}
		}
		a.writeMu.Lock()
		e := a.Store.DB.QueryRow("UPDATE jobs SET status='running',attempts=attempts+1,updated=? WHERE id=(SELECT j.id FROM jobs j JOIN videos v ON v.id=j.video_id WHERE j.status='queued' AND v.deleted=0 ORDER BY j.updated LIMIT 1) RETURNING id,video_id", now()).Scan(&id, &vid)
		a.writeMu.Unlock()
		if e != nil {
			continue
		}
		jobctx, cancel := context.WithTimeout(ctx, time.Duration(a.Config.JobMinutes)*time.Minute)
		e = a.process(jobctx, vid)
		cancel()
		a.writeMu.Lock()
		status := "ready"
		message := ""
		if e != nil {
			status = "failed"
			message = e.Error()
			if len(message) > 500 {
				message = message[:500]
			}
		}
		if ctx.Err() != nil {
			status = "queued"
			message = "Interrupted; will resume on restart"
		}
		a.Store.DB.Exec("UPDATE jobs SET status=?,error=?,updated=? WHERE id=?", status, message, now(), id)
		a.Store.DB.Exec("UPDATE videos SET status=?,error=? WHERE id=?", status, message, vid)
		a.writeMu.Unlock()
	}
}
func (a *App) process(ctx context.Context, id string) error {
	v, e := scanVideo(a.Store.DB.QueryRow("SELECT "+videoCols+" FROM videos v JOIN channels c ON c.id=v.channel_id WHERE v.id=? AND v.deleted=0", id))
	if e != nil {
		return e
	}
	ff := a.helper(a.Config.FFmpeg, "ffmpeg")
	if ff == "" {
		if v.Direct {
			return nil
		}
		return errors.New("FFmpeg is not configured. Original preserved; configure helpers and retry")
	}
	source := filepath.Join(a.Config.MediaDir, "originals", v.Original)
	poster := filepath.Join(a.Config.MediaDir, "thumbnails", v.ID+".jpg")
	input := []string{"-nostdin", "-v", "error", "-y", "-protocol_whitelist", "file,pipe", "-threads", strconv.Itoa(a.Config.Threads), "-i", source}
	if _, e = os.Stat(poster); os.IsNotExist(e) {
		thumb := poster + ".tmp.jpg"
		args := append(append([]string{}, input...), "-map", "0:v:0", "-frames:v", "1", "-vf", "scale=640:-2", "-threads", strconv.Itoa(a.Config.Threads), thumb)
		if e = a.run(ctx, ff, args, nil); e != nil {
			return errors.New("Thumbnail generation failed; original preserved")
		}
		if e = os.Rename(thumb, poster); e != nil {
			return e
		}
	}
	if !a.Config.Transcode {
		if v.Direct {
			return nil
		}
		return errors.New("This video requires transcoding; enable processing and retry")
	}
	tmp := filepath.Join(a.Config.MediaDir, "streaming", id+".work")
	dest := filepath.Join(a.Config.MediaDir, "streaming", id)
	os.RemoveAll(tmp)
	if e = os.MkdirAll(tmp, 0700); e != nil {
		return e
	}
	defer os.RemoveAll(tmp)
	tiers := []int{}
	for _, q := range a.Config.Qualities {
		if q <= v.Height {
			tiers = append(tiers, q)
		}
	}
	if len(tiers) == 0 {
		tiers = []int{v.Height - v.Height%2}
	}
	master := "#EXTM3U\n#EXT-X-VERSION:3\n"
	for _, q := range tiers {
		if q < 2 {
			return errors.New("Video height is too small")
		}
		used, e := treeSize(a.Config.MediaDir)
		if e != nil {
			return e
		}
		budget := (a.Config.StorageMB << 20) - used
		if budget < 16<<20 {
			return errors.New("Insufficient configured storage for streaming renditions")
		}
		if budget > 4<<30 {
			budget = 4 << 30
		}
		name := fmt.Sprintf("q%d", q)
		out := filepath.Join(tmp, name+".m3u8")
		args := append(append([]string{}, input...), "-map", "0:v:0", "-map", "0:a:0?", "-vf", fmt.Sprintf("scale=-2:%d", q), "-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p", "-threads", strconv.Itoa(a.Config.Threads), "-c:a", "aac", "-b:a", "128k", "-ac", "2", "-g", "48", "-sc_threshold", "0", "-max_muxing_queue_size", "1024", "-fs", strconv.FormatInt(budget, 10), "-f", "hls", "-hls_time", "6", "-hls_playlist_type", "vod", "-hls_flags", "independent_segments", "-hls_segment_filename", filepath.Join(tmp, name+"-%05d.ts"), out)
		if e = a.run(ctx, ff, args, nil); e != nil {
			return errors.New("Transcoding failed or exceeded resource limits; original preserved")
		}
		master += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n%s.m3u8\n", q*q*8, ((v.Width*q/v.Height)/2)*2, q, name)
	}
	if e = atomicWrite(filepath.Join(tmp, "master.m3u8"), []byte(master)); e != nil {
		return e
	}
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	var deleted bool
	a.Store.DB.QueryRow("SELECT deleted FROM videos WHERE id=?", id).Scan(&deleted)
	if deleted {
		return errors.New("Video deleted during processing")
	}
	if _, e = os.Stat(dest); e == nil {
		old := dest + ".old"
		os.RemoveAll(old)
		if e = os.Rename(dest, old); e != nil {
			return e
		}
		if e = os.Rename(tmp, dest); e != nil {
			os.Rename(old, dest)
			return e
		}
		os.RemoveAll(old)
		return nil
	}
	return os.Rename(tmp, dest)
}
func (a *App) RetryJob(w http.ResponseWriter, r *http.Request) {
	v, e := a.video(r.PathValue("id"), who(r).User.ID)
	if e != nil || v.Owner != who(r).User.ID {
		problem(w, 404, "Video unavailable")
		return
	}
	_, e = a.Store.DB.Exec("UPDATE jobs SET status='queued',error='',updated=? WHERE video_id=? AND status<>'running'", now(), v.ID)
	a.failure(w, e)
}
func (a *App) Recover() error {
	if e := os.MkdirAll(filepath.Join(a.Config.MediaDir, "incoming"), 0700); e != nil {
		return e
	}
	a.Store.DB.Exec("UPDATE jobs SET status='queued',error='Restarted after interruption' WHERE status='running'")
	rows, e := a.Store.DB.Query("SELECT id,offset,name FROM uploads WHERE video_id=''")
	if e != nil {
		return e
	}
	type pending struct {
		id, name string
		offset   int64
	}
	list := []pending{}
	for rows.Next() {
		var p pending
		rows.Scan(&p.id, &p.offset, &p.name)
		list = append(list, p)
	}
	rows.Close()
	for _, p := range list {
		file := a.incoming(p.id)
		if _, e = os.Stat(file); os.IsNotExist(e) {
			final := filepath.Join(a.Config.MediaDir, "originals", p.id+strings.ToLower(filepath.Ext(p.name)))
			if _, e = os.Stat(final); e == nil {
				if e = os.Rename(final, file); e != nil {
					return e
				}
			}
		}
		if st, e := os.Stat(file); e == nil {
			if st.Size() < p.offset {
				a.Store.DB.Exec("UPDATE uploads SET offset=? WHERE id=?", st.Size(), p.id)
			} else if st.Size() > p.offset {
				if e = os.Truncate(file, p.offset); e != nil {
					return e
				}
			}
		}
	}
	cutoff := now() - int64(a.Config.RetentionDays)*86400000
	a.Store.DB.Exec("DELETE FROM history WHERE updated<?", cutoff)
	a.Store.DB.Exec("DELETE FROM notifications WHERE created<?", cutoff)
	a.Store.DB.Exec("DELETE FROM audit WHERE at<?", cutoff)
	return nil
}
