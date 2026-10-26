# YouTube channel and playlist imports

## Request and starting state

User requests a My Tube feature to save YouTube channels/playlists and download
recent videos. This task does not authorize publication, release or installation.
No Actions, tests or live owner-data mutations were performed. Source checkout:
I:/Projects/my-tube, starting HEAD 760118d. Existing modified protect_other.go and
untracked protect_linux.go came from earlier Linux work and were preserved.
Repository-history and write-aui-ui skills used; read AGENTS before implementation.

## Implementation

- Additive SQLite schema2 in youtube.go; OpenStore accepts existing schema1,
  initializes source/item tables and retains existing identities/media.
- Owner-scoped /api/youtube list/add/settings/sync/cancel/remove endpoints use
  existing authentication, CSRF/Origin checks and write serialization. Strict
  YouTube URL/handle/list/video-ID normalization; no shell/config/cookie imports.
- Saved source gets a private playlist; create a private owner library channel
  only if needed. Sources preserve external attribution rather than creating
  impersonated external-user accounts. 1–50 videos/check, default10, max25 sources.
  Channels read newest regular uploads. Playlists default lastN (new additions
  at end), selectable firstN for newest-first lists; no upload-date-sort claim.
- One server background importer. Manual checks default; optional daily checks
  run while server runs. Per-source and per-item status, byte progress, failures,
  retry, cancel, source removal. Source scanning5min, whole batch12h, per-video
  configured JobMinutes. Rate limiting stops the batch. Saved videos survive
  cancellation/restarts; incomplete feature-owned reservations are cleaned up.
- Video IDs deduplicate within owner across saved sources; hashes catch existing
  originals too. Normal begin/finish path reserves quotas, verifies container,
  applies optional scanner and publishes atomically to configured media_dir.
  Existing HLS workers prepare playback. No Drive permissions bypass; existing
  scoped export remains available. Source removal keeps playlist/videos. Deleting
  its playlist removes the source, with active-job deletion guard.
- Up to720p H264/AAC. With FFmpeg/FFprobe, use separate HTTPS streams and FFmpeg
  stdout fragmented MP4 directly into a bounded writer (no full merge scratch).
  Existing automatic media tools prepare on Windows. Without helpers, select a
  combined MP4, potentially lower quality. Availability depends on YouTube.
- First explicit import downloads pinned upstream yt-dlp2026.10.21 and Deno2.9.8
  for Windows/Linux x64. Archive/API/executable digests pinned in youtube_tools.go;
  Deno executable hashes derived from verified official archives. No helper is
  redistributed in the app. HTTPS host/size/time bounds, cache hash verification,
  known-file-only repair. Runtime scratch within app helpers, not host TEMP.
  Windows bounded kill-on-close job allows downloader/Deno/FFmpeg children;
  Unix process-group cancellation. This is not a hostile-code sandbox.
  Linux cached helper execute bits are restored only after digest verification,
  because existing pending Linux data protection resets ordinary files to0600.
- Tube.aui / Dialog.aui generated through existing compiler. New Imports page,
  sidebar and upload entry, settings/dialog actions, responsive cards using
  existing components/tokens. Unknown download size uses indeterminate progress;
  saving/completed items have no active bar. Dialog mutations disable while busy.
- README/NOTICE and docs/YOUTUBE-IMPORTS.md explain workflow/provenance/limits.

## Evidence and remaining work

Frontend AUI generation, TypeScript and Vite production build passed. Native
Windows x64 and cross-compiled Linux x64 builds passed. Existing Vite >500kB
bundle warning remains; build does not establish browser/network acceptance.
No tests added/run (current user request did not ask for testing). No executable
was started, no real YouTube download/browser workflow, schema migration runtime,
restart/cancel or Linux execution was exercised. These remain acceptance gates;
do not describe this implementation as live-tested or installed.

Builds/downloaded tool-pin evidence are on I:/Projects/tooling/tube-youtube-20261025.
No new C-drive fixture/install/archive was created. Source changes are uncommitted;
no push, release, update-index publication or owner update/install in this task.
Version remains the existing preview2 pending a separately authorized release.
Schema2 databases require this or a newer build; use pre-update backups when
rolling back to schema1. Existing Linux protection changes must be reconciled
with their original work before any publication, not silently discarded.

## Publication authorized

User now requests push, release and install on the other machine. Prepare
preview3, cross-platform archives and the current Workspace-compatible app
manifest; preserve manager90/other app versions on the secondary. Current org
main still760118d before this task. Actions disabled. Review retained Linux
protection implementation and include it with the Linux build; restore executable
bits for verified cached tools. Raise Go directive/toolchain to1.27.3 because
the current suite records the1.27.2 security rebuild. Current secondary is90,
not the historical80. Another task owns an updater tab; use a separate tab and
respect the shared operation lock. MyAI7 is separately pending; do not select it.
Publication, public channel promotion and secondary completion pending here.

## Preview3 publication complete; secondary installation waiting

Published v0.1.0-preview.3 from ed1e46cd746228ab9f56bbc2cd2115a4b153c9d8,
locally built Go1.27.3. Windows/Linux standalone archives, sources, notices,
standalone and managed manifests, build information and checksums: all13 public
asset sizes/digests match local. Anonymous Windows ZIP download matches SHA256
3f4ff41c2ead34ea169a9519df4df5b06dff37bd6934055ff1ebf299ee39a323.
Workspace channel/pin promotion c31b168 is pushed and raw public main contains
preview3; every earlier promoted tag is retained, including withheld decisions.
No Workspace manager release was created; existing90 component contract retained.

Secondary normal updater is already applying an independently requested MyAI7
transaction. Do not interrupt, modify it or replay apply. Fresh separate public
updates tab reconnects; Remote Desktop current installed90 local controller
shows progressing8.6GB backup. Wait for that transaction to finish, then check
again and select Tube only. Old local71/75/77 browser tabs are closed-controller
history and must not be treated as current. Opened existing installed updater
with normal --data updates command to view current operation, not start one.
Remote on-screen input restored off; clipboard preference unchanged. No native
fixture or owner import performed. Final Tube installation still required.

## Secondary Tube3 operation accepted

Independent MyAI7 operation completed healthy in32m26s. Fresh anonymous-channel
check on secondary90 shows MyAI7 current and exactly one available checkbox:
My Tube2 ->3. Captured displayed pre-update version rows, then submitted exactly
one normal Back up and install selected updates action for Tube only. Download
and package checks passed; operation reached storage/restart checking. Do not
replay apply if its public tunnel disconnects. Wait for backup/install/startup/
cleanup completion, then fresh check and open Imports to finish. Other25 displayed
update groups are the baseline; My3D's existing metadata-review error is unrelated.
Remote on-screen input was temporarily used a second time after Chrome closed to
reopen the existing90 controller via normal updates command, then restored off.

## Delivery complete

Secondary normal Tube-only update completed healthy in23m28s. Final fresh public
release check reports Tube0.1.0-preview.3 up to date; all25 other displayed version
rows exactly match the pre-Tube baseline, including Workspace90/MyAI7 and the
pre-existing My3D metadata-review warning. Newest verified recovery retained;
normal cleanup removed one older automatic copy (~9239.1MB). Exact private receipt
and backup path are outside the repo in tooling/tube-youtube-20261025/
secondary-install3.json. No update applied to the primary by this task.

Installed app opened signed in without credentials and served preview3. The new
Imports page loaded its empty source list without error; screenshot retained
outside the repo as secondary-tube3-imports.jpg. This verifies installation,
startup, version and basic Imports UI/API, not actual YouTube download, cancel,
scheduled sync, Linux runtime or full security acceptance. No owner videos were
imported as fixtures. Automated tests remain unrun. No Actions used.

Public sources/release/channel are pushed. Large artifacts and evidence remain
on I; no new C fixture/install/archive. No task fixture processes remain. Remote
on-screen input restored off and clipboard setting unchanged. Normal installed
updater is intentionally idle/current. No remaining required release/install work.

Operational note: existing cleanup reuses the retained-copy progress label while
verifying an older copy before retirement; file totals may reset during stage9.
This was normal progress, not a repeated apply or failed installation. Public
tunnel loss/Cloudflare1033 during snapshot recovered after restart; exactly one
Tube apply was used. Source tag3 remains ated1e46c; later handoffs are docs only.

## Import repair after preview3 delivery

Owner reported all five selected public videos failed and expected a channel
page rather than only a playlist. Continued the authorized Tube release/install
work. Extraction with the exact pinned tools returned11,621,421bytes of full
video JSON, exceeding the app's2MiB writer; the process error was hidden by the
generic message. Projecting consumed fields returned1,834bytes successfully.
The request now also includes duration for verification.

Preview4 adds private source-backed channel pages/owner sidebar entries, name,
about/artwork, discovered pending/failed items, saved video browsing, retry and
refresh. Additive schema3 backfills channel sources, preserving creator identity,
video ownership and media ACLs. Artwork is bounded/re-encoded JPEG/PNG from fixed
YouTube image hosts with redirect checks. All selected items appear before the
first download. Private playlists/deduplication remain. Default imports select
480p or lower, including existing sources. Errors distinguish metadata limits,
HTTP refusals, unavailable formats and interrupted connections.

Actual downloader diagnostic also exposed FFmpeg reporting success after a TLS
reset: video522.47s/audio446.39s versus source524s. Added bounded HTTP reconnects,
-xerror and per-audio/video-duration verification before ordinary publication.
That diagnostic was not a complete saved video. Automated tests were not added
or run. Frontend generation/TypeScript/Vite and Windows/Linux builds passed before
the final bump; final preview4 builds/packages/publication in progress. Complete
installed repair import, Linux runtime, clean-host/reboot acceptance are pending.

Artifacts are on tooling/tube-import-fix-20261026 (folder date is a task name).
No C fixture and no live owner library used as disposable test data. Source3
release stays immutable. Next: publish4, promote Workspace pin/channel, install
Tube only on secondary and retry the existing requested source. Preserve90,
MyAI7/other versions, operation locks and newest protected rollback.

Preview4 publication complete: source948db4a, tagv0.1.0-preview.4; all13 asset
sizes/digests verified. Anonymous public Windows ZIP hash/size match; public raw
channel promotionad5dea0 and future suite pin pushed. Prior tags/withholding
preserved. Secondary Tube-only normal update is next; no manager91/full installer
replacement. Raw diagnostic signed-URL metadata and partial media removed from
this task folder on I; no C scratch created.

Secondary preview4 update submitted exactly once after the fresh public channel
check offered only Tube3->4. Download/package checks passed; current public UI
shows stage4 compatibility/restart checking then connection loss. Do not repeat
apply or recover while its normal protected operation runs. Public source/release
and channel are delivered; installation and existing-source retry pending.

## Preview5 quality correction while4 installs

A focused format listing showed the source video offers480p only as AV1 MP4;
H264 is360p/720p. Preview4 correctly caps480 but falls back360 with that codec
filter. Preview5 prefers H264 at480, otherwise keeps the best MP4 up to480 when
normal processing is enabled (including AV1), with normal H264/AAC HLS generation.
When processing is explicitly disabled it retains compatible H264 selection.
Do not replace published4 assets or alter its active installation transaction.
Prepare/publish/promote5 separately; finish4 first, then normal5 update if needed.
No source/helper pins, media ACLs or other app contracts changed.

Preview5 published fromb0dacbf; all13 assets verified. Public raw refs/heads/main
contains5; anonymousWindowsZIP matchesd23642d6f1361b078380bfec156f9e6b3463bd95cc41aa95e158aee2d5a99e10.
Promotion6f64a0f rebased onto concurrent823aab1; regenerated index preserved all
remote-main releases and MyAI8 future-suite pin. Bare main raw path temporarily
serves cached4; wait for normal updater5 availability. Preview4 active operation
reached stage8 restarting/health-checking at18m32. Do not interrupt it or apply
MyAI8 on this task. Need finish4, normal Tube5-only update, then existing source
retry and verify saved/playable480p plus channel page. No complete import yet.

Preview4 normal installation completed23m38s, all apps healthy. Cleanup retired
one older automatic backup (~9346.5MB) and retained newest usable rollback.
Fresh normal anonymous-channel check confirms Tube4 installed and5 available.
Original source retry remains pending; portal sign-in expired after restart.

User separately approved a combined MyAI/Tube update in Ai chat. Verified that
human reply with read_thread before coordinating; acknowledged there. Selected
exactly MyAI8 plus Tube5 in this task's secondary updater, then submitted one
normal protected apply after4 fully completed. MyAI source4b97923, expectedEXE
4bade35628956b0b1caf13cdc415759e5f25854245e1f93a826ead5e2abf565a;
Ai task independently verified its publication/primary deployment. No primary
operation here. Preserve manager90/all24 other displayed installed groups,
models, identities/settings and newest rollback. Before-combined rows retained
in browser variable tube5BeforeCells. Finish combined operation; report completion
to Ai task under verified human coordination approval. Private sign-in may be
needed after the final restart before retrying the existing requested Tube source.
No complete installed import claim, owner-library fixture, Actions or new tests.
