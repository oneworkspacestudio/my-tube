# Optional media helpers

The main app plays conservatively verified H.264/AAC MP4 without a helper.
Its bounded ISO-BMFF parser checks box sizes, movie metadata and AVC sample
configuration, rejecting unknown/encrypted/fragmented layouts. FFprobe is
required for broad MOV/MKV/WebM support and additional stream analysis. MP4
with unsupported codecs is retained and needs FFmpeg for browser playback.

On Windows x64, optimisation and automatic tools are enabled by default. The
first launch downloads the pinned FFmpeg9.0.2 essentials archive directly from
[Gyan's upstream release](https://github.com/GyanD/codexffmpeg/releases/tag/9.0.2),
a Windows supplier linked by the [official FFmpeg download page](https://ffmpeg.org/download.html).
The archive is114,768,076 bytes; its SHA256 and separate FFmpeg/FFprobe executable
hashes are compiled into the app. Neither program runs before both checks pass.
Settings shows actual byte/percentage progress. Tools are separate processes
cached in private data/helpers/ffmpeg-9.0.2. My Tube's EXE and ZIP do not
redistribute FFmpeg binaries. Source/licensing links are in media-tools.json.

New uploads and existing completed originals without the current profile are
queued automatically. Failed downloads retry at one-minute intervals. Explicit
transcode:false is respected; omitted settings get the new true default. Set
auto_helpers:false to disable downloads, or configure reviewed tool paths.
Originals stay playable and unchanged on failure. Keep the app running until
Creator Studio shows processing complete.

The encoder must include libx264 and AAC. Outputs are H.264 yuv420p + AAC stereo,
VOD HLS MPEG-TS with four-second segments, aligned two-second keyframes, 30fps,
and a master playlist with measured peak/average bandwidth. Target video rates
are450/800/1500/3000kbps for360/480/720/1080p; capped peaks are650/1100/2000/4000kbps,
plus96/128kbps audio. Auto starts with the smallest tier and upgrades
conservatively, capped to the player size. Buffer memory/time are bounded. Renditions are limited to configured
360/480/720/1080 tiers at or below the source height. Smaller inputs use an even
source height instead of upscaling. hls.js provides adaptive playback where MSE
is available; browsers with native HLS can use that path. WebVTT captions can
be uploaded in video details. Custom images are decoded and re-encoded to JPEG.

Limits: 24-hour source duration, 16384-pixel source bounds, fixed worker/thread
caps, 1 GiB helper memory on Windows, helper process timeouts, bounded output
capture, file/pipe protocol allowlists and a media-storage monitor. These are
resource controls, not a decoder sandbox. Use current helpers and least privilege.
Set `processing_hours_utc` to schedule processing in quieter hours, or keep
automatic transcoding off on low-powered hardware.

FFmpeg licensing is build-dependent: the project is generally LGPL, while
optional GPL components such as libx264 make a build GPL. Never redistribute an
`--enable-nonfree` build. If distributing helpers in your own installer, supply
the correct license notices, corresponding complete source/build configuration
and required redistribution rights for that exact binary. Review the
[official FFmpeg legal guidance](https://ffmpeg.org/legal.html). This release
ships no FFmpeg binaries, DLLs, codecs or source archives.
