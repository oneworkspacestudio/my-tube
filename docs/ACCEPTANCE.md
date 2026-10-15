# Preview acceptance

The preview has a working native application. It is not designated stable until
the remaining deployment matrix is completed.

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
