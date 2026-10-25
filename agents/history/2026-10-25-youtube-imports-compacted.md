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
