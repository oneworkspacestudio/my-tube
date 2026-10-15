# Optional media helpers

The main app plays conservatively verified H.264/AAC MP4 without a helper.
Its bounded ISO-BMFF parser checks box sizes, movie metadata and AVC sample
configuration, rejecting unknown/encrypted/fragmented layouts. FFprobe is
required for broad MOV/MKV/WebM support and additional stream analysis. MP4
with unsupported codecs is retained and needs FFmpeg for browser playback.

Obtain a reviewed, current Windows FFmpeg build through the links on the
[official download page](https://ffmpeg.org/download.html), verify the supplier’s
checksums, and put ffmpeg.exe and ffprobe.exe in `helpers/` next to MyTube.exe,
or set their absolute paths in configuration. Keep any DLLs required by that
particular build beside its executables. FFmpeg is not automatically downloaded
or redistributed by My Tube. Use `transcode:true`, restart, and retry queued or
failed jobs from Creator Studio. Existing compatible MP4 files stay playable if
the helpers are missing.

The encoder must include libx264 and AAC. Outputs are H.264 yuv420p + AAC stereo,
VOD HLS MPEG-TS, and a master playlist. Renditions are limited to configured
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
