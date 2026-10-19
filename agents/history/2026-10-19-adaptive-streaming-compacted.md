# Adaptive playback repair

User authorizes source optimisation, org push and Windows release; no GitHub
Actions. Source at I:/Projects/my-tube; org main4dab3a4 at start. App Actions have
been disabled explicitly before any push; older preview1 CI history is not a
workflow to follow. Work uses repository-history and AUI-generated screen trees.
No live owner data is a functional test fixture or modified by this task.

Diagnosis handoff records original-only playback (about20.6Mbps); processing
default was false. Implemented default-on transcoding, default automatic tools
on Windows x64, bounded450/800/1500/3000kbps video tiers,650/1100/2000/4000kbps
maxrate and96/128kbps AAC. Four-second HLS, aligned two-second keyframes,30fps,
bounded threads and no upscaling. Master peak/average bandwidth is measured from
the actual TS segments, not guessed from height. Original files are preserved.
Auto starts at lowest tier, uses conservative bandwidth/headroom and player-size
capping; bounded buffers. Transition to prepared HLS preserves playback position.

Windows automatically fetches Gyan's immutable FFmpeg9.0.2 essentials archive
directly from upstream, verifying archive SHA256/size and both executable hashes
before use. Pinned metadata is in internal/app/helpers_pin.go and docs/media-tools.json.
App distributions contain no FFmpeg binary. Private data/helpers cache is checked
at startup; only known cache files can be repaired, unknown files/junctions are
preserved/rejected. Downloads have host/redirect/size/time bounds and fixed output
names. Settings shows real byte/percentage progress; offline failures keep the
original playable and automatically retry at one-minute intervals. Explicit
transcode:false / auto_helpers:false and reviewed custom tool paths are respected.

Existing completed originals without the current profile are queued on startup.
Completed profile/master pairs are reused. Profile/outputs publish only after a
successful bounded job. No overwrite of source. Whole-host helper and media limits
remain; this is not a hostile decoder sandbox or a zero-buffering guarantee.

Go tests/vet passed with real reviewed FFmpeg/FFprobe9.0.2, including existing
visibility/range/helper-failure checks and new bounded-ladder/measured-bandwidth/
safe-segment/profile queue regressions. Frontend TypeScript/Vite/AUI build passed.
Seven native synthetic checks upgraded the published preview1 executable's
existing private1080p original to local preview2: automatic real upstream helper
download, all four renditions, original SHA/ranges unchanged, all private media
routes reject anonymous access, actual Chrome HLS decoding under CDP1.2Mbps,
and cache/rendition reuse across restart. Source21.38Mbps; measured average tiers
0.586/0.946/1.697/3.231Mbps; throttled playback decoded640x360 beyond7seconds.
An initial harness channel request omitted visibility; corrected request (no
product change). No remote tunnel, clean-host/reboot/service or independent audit
certification is implied by a simulated network check.

Next: commit final source, local release build My Tube preview.2, final packaged
smoke, component manifest/source/provenance and verified GitHub publication.
Coordinate Workspace preview.41 with exact published component SHA and new managed
defaults; preserve explicit existing settings. Other component pins unchanged.
External scripts/evidence/binaries under I:/Projects/tooling/tube-streaming.
Protected disposable state only C:/Projects/.cache/workspace-tube-streaming-20261019.
Track/stop every fixture, check process/path/reparse boundaries and remove that
exact C root after gates. Owner C:/tmp/workspace/{Apps,Private,Files} must stay
untouched. Source/artifacts stay on I:. Append actual publication/cleanup receipts.
