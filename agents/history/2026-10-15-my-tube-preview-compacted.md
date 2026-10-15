# My Tube implementation — 2026-10-15

## Scope and source

User requested a new `oneworkspacestudio/my-tube` app implementing the supplied
My Tube master brief. The brief authorizes org repository creation, source
publication and Windows release. It is preserved at docs/PROJECT-BRIEF.md.
Authoritative checkout is H:/Projects/my-tube. No existing Workspace installation,
MyAccounts data or Drive library was changed. Repository creation succeeded.

Use repository-history at the start and end of material repo work. UI trees are
web/src/Tube.aui and Dialog.aui, compiled using @codedia/parser 0.2.1 plus a
semantic React adapter. Regenerate; do not hand-edit generated TSX. Build web
before Go embeds it. Data tests must use disposable folders.

## Implemented architecture

Single CGO-free Go main executable MyTube.exe, pure-Go modernc SQLite/WAL/FTS5,
embedded React/TS/Vite/Tailwind/Lucide/TanStack Query frontend and hls.js.
No Docker, Node, Python or DB server runtime. FFmpeg/FFprobe are optional separate
helpers and are not redistributed; licensing boundary is docs/MEDIA.md.
AGPL-3.0-or-later; hardened OIDC/session/Windows ACL foundations adapted from
the organization Notes app with attribution. No Joplin dependency or Google art.

Uploads reserve quotas, use synced 8-MiB chunks with persisted offsets, verify
containers/streams, hash duplicates per owner, rename atomically and publish
metadata/jobs transactionally. Conservative bounded ISO-BMFF analysis permits
H.264/AAC MP4 without helpers. FFprobe analyzes wider formats. FFmpeg generates
posters and HLS at configured <=source tiers. Persistent workers have bounded
concurrency/threads/time, Windows Job Object memory/process caps and storage
monitoring. Failure preserves originals. Recovery requeues jobs/trims unacked
chunks/restores interrupted renames. Optional UTC processing hours.

Channels/Creator Studio, metadata/images/captions/chapters, public/unlisted/private
and explicit private viewers, likes/subscriptions/notifications, comments/replies,
Watch Later/playlists/history/resume, FTS search and explainable recommendations.
Preference disables personalized signals. Lightweight analytics distinguish
retained authenticated viewers and approximate client-reported watch aggregates.
Replacement publishes a verified new upload under a new URL and retires the old
record; old original remains until explicit admin purge. Social history stays on
the previous record. Administration config applies after restart. Admin can
explicitly check published GitHub release versions; binary installation is
manual. Drive browsing offers clickable file/folder selection before import.

One visibility predicate governs metadata and ALL originals/posters/HLS/segments/
captions. No app-admin playback bypass. Public settings default off. OIDC is code
+ PKCE with metadata, state/browser binding, nonce, signature/issuer/aud/expiry.
Opaque hashed sessions, CSRF and exact Origin, fixed expiry and local logout.
Managed Windows data/media require ACL-capable volumes; source H: exFAT is fine,
runtime managed data must be on NTFS. First-run/local URLs are one-use loopback
links. Default standalone launch opens setup, explicit --local skips it; launcher
OIDC env skips setup. Admin subjects are explicit, never first-user promotion.

Quantum storage adapter uses actual /api/users?username=self, /api/resources and
/api/resources/download with the caller's cookie/scoped token. Effective admin
tokens and redirects are rejected; no shared admin credential. Imports are
explicit owned snapshots because external ACL revocation cannot safely govern
unmanaged filesystem references. Source is unchanged; export never overwrites.
No direct shared-directory shortcut is enabled. Full details docs/INTEGRATION.md.

Offline CLI backup uses SQLite snapshot + all media + SHA256 manifest. Restore
requires fresh data/media, validates paths/hashes/DB/media relationships, clears
sessions/logins, never overwrites an existing DB. CLI run/install/uninstall/start/
stop/version/config-check/health/migrate, service identity docs and instance lock.
Launcher manifest has actual executable digest/size and /my-tube route/PKCE.

## Verification and tools

15 Go tests passed including upload/resume/quota/type/CSRF, all private media
routes, social/FTS/ownership, replacement, restore/migration guards, prefix config,
real FFmpeg HLS/poster/failure recovery, OIDC forged signature/wrong issuer/expired
token/nonce/audience/replay and a native ACL Everyone-removal test. Go vet and
govulncheck passed. npm audit reports zero vulnerabilities at this checkpoint.
TypeScript + production frontend and native Windows build passed.

Six installed Chrome workflows passed with optional helper paths: channel/upload/
real video decoding/social, HLS decoding/quality, playlists/search/preferences,
anonymous private derivatives, desktop/mobile/dark screenshots, first-run wizard.
No horizontal overflow observed. Screenshots use the self-generated test MP4.
Native forced restart/resumable chunks, duplicate lock, portable DB/media, offline
backup/restore/digest and /my-tube HTML/assets pass via test-portable.mjs.

Real disposable MyAccounts executable preview-20261014.2: two normal users
complete actual provider code/PKCE/token/JWKS, retain stable subject mapping and
private-video separation/logout. scripts/test-myaccounts.ps1 and corresponding
web script create fresh NTFS/DPAPI fixture and stop only owned processes.
No existing credentials read or changed. MyAccounts HTTP fixtures require
localhost, not 127.0.0.1 as issuer.

Real disposable FileBrowser Quantum executable from
I:/Projects/filebrowser-drive/build/drive-upload-preview/FileBrowser.exe: fresh
password fixture, nonadmin user, resource listing/import/export/digest/no-overwrite/
original preservation/private denial pass via web/scripts/test-drive.mjs.
The first readiness probe incorrectly used nonexistent /api/config; corrected to
the real root page. Supplying a helper directory avoids Quantum’s default helper
setup delay. Test configuration and credentials never enter the repository.

Host Go: I:/Projects/tooling/go1.27.2/go/bin/go.exe. Caches can use
H:/Projects/tooling/notes/{go-cache,tmp,npm-cache}. MYTUBE_ACL_TEST_ROOT must point
to NTFS when GOTMPDIR is exFAT (Go1.27 TempDir follows GOTMPDIR).
FFmpeg/FFprobe dev fixture: host Anaconda 4.2.2, never shipped or recommended for
production. Chrome is installed at the standard Program Files path. UI checkbox
adapter uses immediate local state to bridge query-cache subscription scheduling.

## Publication and open gates

Version 0.1.0-preview.1 is published as a public prerelease with ten verified
assets. Tag points to source `87895a5ab392cecdbac1167d544a035c195ef03c`.
Release: https://github.com/oneworkspacestudio/my-tube/releases/tag/v0.1.0-preview.1
Final hosted Windows CI passed:
https://github.com/oneworkspacestudio/my-tube/actions/runs/38813374411
Earlier CI also passed at fc97241. Final CI builds/tests/vets/scans/packages the
executable, five browser workflows and portable restart/restore; optional real
helper/provider fixtures run on the development host. Workflow actions use
immutable SHAs, dependencies are pinned, and newer runs cancel superseded ones.

The RELEASE executable and ZIP are downloaded from that successful final CI,
not the local development build. Exact CI MyTube.exe is 14,213,632 bytes, SHA256
`cdce29f4b223c6d86cdeb0b01532d6f5438c5dbe3f4899a7abdab2a79dbffa83`.
Portable ZIP is 6,500,346 bytes, SHA256
`3ef6f9d8e9f9d5f1ec25bff59561e4ce6f7d998365847c71d816facb248ab0b8`.
All ten GitHub-hosted asset digests/sizes, tag target and draft=false/prerelease=true
were checked. ZIP inner executable matches the manifest. CI files are retained
at H:/Projects/tooling/my-tube/release-ci-87895a5; dist/ is a development build.

The exact CI executable additionally passed all six local Chrome workflows with
real HLS helpers, the disposable two-user MyAccounts fixture, the real nonadmin
Quantum import/export fixture, and an explicit update check against the newly
published GitHub release. Visual tests disable finite transitions before capture;
otherwise an immediate dark-theme screenshot caught the 150-ms chip transition.
No app theme defect was inferred from that intermediate screenshot.

SSH is required for workflow-bearing pushes because HTTPS OAuth lacks workflow
scope. No credential changes were needed. Post-publication handoff updates use
a docs-only [skip ci] commit; they do not change the release tag or executable.

Stable-ready=false. Remaining acceptance: fresh Windows10/11, service/reboot and
upgrade/rollback, actual launcher add-app/browser OIDC at deployed origin, live
shared-origin Drive cookie behavior, public Cloudflare/reverse-proxy limits,
mobile Safari/native HLS matrix, code signing and independent security review.
Preview limits and exact evidence are in docs/ACCEPTANCE.md. Large libraries need
pagination beyond 500 discovery candidates; synchronous Drive writes serialize
with other mutations. No stable/live deployment claim is justified by fixtures.
