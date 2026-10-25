# Preview acceptance

## YouTube import preview 3

This version adds YouTube source tracking/downloads and Linux x64 packaging.
Frontend generation, TypeScript/Vite and Windows/Linux x64 compilation pass.
No new automated tests or real YouTube download workflow was run for this change.
Historical checks below apply to the versions named there. This is a prerelease;
do not read the older playback/auth checks as YouTube import acceptance.
Builds use Go 1.27.3. Helpers remain separate pinned upstream downloads.

## Earlier preview evidence

The preview has a working native application. It is not designated stable until
the remaining deployment matrix is completed.

The [hosted Windows build](https://github.com/oneworkspacestudio/my-tube/actions/runs/38813374411)
passed at source `87895a5ab392cecdbac1167d544a035c195ef03c`. Release assets come
from that CI artifact. Its exact executable additionally passed all six Chrome
workflows with helpers, the real MyAccounts and Quantum fixtures, and a check
against the published release-version API. Ten hosted asset hashes/sizes and
the release tag target were verified. The release remains a prerelease.

Verified on the development Windows host:

- Go store/API/auth/security/recovery tests, Go vet and govulncheck.
- Production TypeScript/AUI/Vite build and dependency audit.
- Six Chrome workflows (one requires optional helpers): channel creation, file upload, actual video playback,
  metadata edits, persisted comments, Watch Later/playlists/search/preferences,
  anonymous private media denial, mobile overflow, light/dark screenshots, first-run configuration and real HLS decoding.
- Real FFmpeg/FFprobe: MP4 analysis, poster, 360p HLS, no upscaling, per-file
  derivative authorization and failure preserving the original.
- Real disposable MyAccounts server: discovery, code+PKCE, signed identity,
  stable account mapping, two-account private separation, logout.
- Real disposable Quantum server with non-admin user: scoped listing, original
  import/export, byte integrity, no overwrite and private playback denial.
- Native forced process restart, partial upload resume, database/media survival,
  duplicate-instance lock, offline full backup/restore and original digest match.
- Workspace `/my-tube` prefix HTML, asset and API behavior.

Fixtures use self-generated media and fresh data; no existing Workspace, identity
database, Drive library or user content is modified. Optional integration tests
are reproducible scripts. Current development media helpers are FFmpeg 4.2.2
from the host’s existing tool installation; they are **not** shipped or recommended
as a current production helper build.

Remaining stable gates: fresh Windows 10/11 launch with only the release files,
service identity/install/start/stop/reboot, clean-host upgrade/rollback matrix,
real launcher add-app registration and browser sign-in at its deployed origin,
live shared-origin Drive cookie behavior, public reverse proxy/Cloudflare load
and limits, independent security review and code signing. Local fixtures and CI
do not establish those live deployment outcomes. Native HLS and mobile Safari
require additional browser-matrix testing.

Intentional preview boundaries: one channel per user; replacement creates a new
URL; analytics are approximate client-reported aggregates and unique counts only
cover retained authenticated viewers; imports are explicit owned snapshots;
configuration applies on restart; no automatic binary self-update; no email/push
notifications, distributed transcoding, shared-directory ACL mapping or end-to-end
encryption. Large catalogs need pagination/throughput work beyond the 500-candidate
list cap. Browser uploads run sequential chunks with concurrent file tasks; the
server serializes writes to preserve quotas. Do not present these as unbounded
public hosting, a fraud-resistant analytics system, or a fully isolated decoder.
