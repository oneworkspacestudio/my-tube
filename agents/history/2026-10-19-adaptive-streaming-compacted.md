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

## Component publication and final native gates

Published v0.1.0-preview.2 from da3dc1d8f59d9932c09cac71b3528c484c9cdf6f.
All ten remote asset sizes/SHA256 matched; the public ZIP was downloaded/hashed
(bf788ebef31d770224eb7d548d7c771e177c537707330ebcf0ef8d4521c6a5ac).
The final packaged EXE separately passed original-byte preservation, cache reuse
and simulated1.2Mbps playback. Actions remain disabled and were not used.

Workspace41 source17a37ac builds an exact19-app bundle whose My Tube packages
match this published component. Native isolated baseline40 with Accounts/Drive/
MyTube1 was seeded with a synthetic original, then the released40 updater applied
Tube2 and the41 installer updated the manager. IDs/bytes/identity/paths survived;
actual managed SSO and360p playback at simulated1.2Mbps passed after updating.
Task-owned processes/browsers stopped; named C fixture removed (~1.62GB), with
source/evidence on I:. Owner install/data/tunnel untouched and local health200.
Workspace41 is published from17a37ac1f98c514593805e390b9b44b5df5435cc, with24
verified remote size/SHA256 assets, downloaded/hashed public catalog and anonymous
installer/bundle availability. Tag source matches build provenance. Both org
main branches contain implementation; no Actions used. Publication complete;
owner must update Workspace/MyTube and leave it running while originals prepare.
Do not retag or replace immutable published assets.
