# YouTube schedules and rolling retention

## Request and baseline

User requests background latest-video checks for imported channels, default10
saved videos with20/30/50/Unlimited options and older copies removed. Continuing
the My Tube app update/release flow authorized in this chat; both PCs have6.
Started from clean3025688, equal to origin/main. No owner data is a test fixture.
Repository-history and AUI skills applied. Version7 is ready for clean-source
packaging; public release/channel and installed delivery remain pending.

## Implementation

- Schema4 adds retention_limit (existing sources0/Unlimited), observed source
  positions, proven importer-owned media and a durable purge queue. Existing
  schedules/scan windows/copies survive migration. New imports default10/daily.
- Manual/hourly/6-hourly/daily/weekly schedules persist and run on the server.
  Completion/failure/cancel reschedules from finish; missed checks resume after
  startup. The browser need not remain open; the Workspace server must run.
- Finite retention scans at most its limit and keeps successfully linked videos
  by observed source order. Failed replacement downloads do not evict old media.
  Playlist-end order is handled separately from newest-first channel order.
- Real upload finalization records ownership only when a feature-owned upload
  created the original; hash-deduplicated ordinary uploads never acquire it.
  Unknown legacy ownership is not retroactively inferred. Expiry detaches those
  library copies from the source but preserves their originals.
- Expiry retains other-source/playlist/Watch Later/shared/public references.
  Proven unreferenced managed copies become inaccessible atomically with source
  unlinking, then original/HLS/poster/subtitle bytes are purged. Processing and
  Windows playback locks defer deletion. Queue survives restarts. All paths and
  ancestors reject symlinks/reparse points/special files before deleting bytes.
  Removing a source releases its automatic ownership and keeps saved media.
- Settings are unavailable during an active import; cancel/finish first. UI uses
  existing AUI/dialog adapters and adds Schedule & storage plus retained/pending
  counts. Unlimited keeps all downloaded copies but scans up to50 recent entries
  per check; ordinary storage quotas remain. It is not full historical backfill.

## Verification and next steps

Eight focused synthetic regressions cover defaults/options/owner isolation,
rolling replacement and physical derived-file cleanup, protected references and
ordinary uploads, Unlimited, deferred processing cleanup/restart, actual worker
catch-up/manual scheduling, schema3 migration, playlist ordering/source removal,
and linked/traversal storage rejection. Full Go suite/vet and production frontend
build pass. Windows/Linux builds and notices run in task artifacts; visual checks
are next. The initial protected-playlist test expected201 instead of the existing
API's200; corrected the assertion, with no production behavior workaround.

Artifact root: tooling/tube-retention-20261103; named NTFS synthetic root:
C:/Projects/.cache/tube-retention-20261103. check-ui.cjs seeds only a stopped
synthetic local database, with future/manual sources to avoid real downloads.
No real YouTube or owner imports were used for these tests. Verify UI/builds,
review final diff, package immutable7 and promote through the supported channel
and managed update paths. Reuse6 publication patterns, not immutable6 assets.

PC2 coordination: an unrelated owner My AI benchmark resumes after the6 update.
Check its current state before another PC2 restart; do not interrupt an active
benchmark or replay an in-flight update. No existing source retention/schedule
should be changed as a deployment fixture. Record actual target receipts and
cleanup before claiming7 delivery complete.

## Candidate acceptance

Final Windows and Linux cross-builds and notices completed. Headless Chrome on
an isolated native Windows /my-tube/ instance verified all five retention options,
all five schedules, 10/daily defaults, retained legacy settings, real API saves,
restart persistence and phone/desktop layout. Four synthetic previews retained
under docs/screenshots/youtube-retention-*.png. No YouTube command ran in this
fixture. A first UI pass timed out only because its restart check still used a
phone viewport with the desktop sidebar hidden; the check now restores desktop
width before that navigation. Production UI was not changed to mask the fixture.
Cleanup work is bounded to four videos per pass and retries every two seconds.
The final schema3 migration test also preserves an existing video's playlist
reference without inferring importer ownership. Publication, public channel and
installed7 verification remain pending. The separate PC2 model benchmark now
visibly reports Finished; no benchmark was interrupted by this task.

## Immutable publication and channel

Source cd3c8cc5c45e21b04a2a8fca7ff386cd282ae64f pushed to main/tag7. All13 public
assets verified against local sizes/GitHub SHA256. Anonymous Windows ZIP hash
288195b4fec58d72eca62bebf9358087a59dc5652d53f6371c56d31ee48a9de6;
EXE59e1d82a7a3508915dcfb508582e12e221b693318f2f781fae8a978144a6d2ec.
Actual candidate planner gates pass in both catalog layouts, plus component
projection retaining all36 unrelated app contracts/447 package contracts.
Workspace public pin/channel e3919c9 pushed; every preceding row/asset and all
withholding records identical. Exact raw-main CDN still caches the prior snapshot
for five minutes; wait for normal anonymous discovery before applying. Full
ordinary build running locally. No suite release/tag or Android claim.

Synthetic NTFS fixture and its processes cleaned up. Both owner targets still6;
fresh updater baselines collected, no7 apply yet. Finish normal backed-up My Tube-
only updates and verify receipts, fresh up-to-date discovery, unrelated versions
and served feature UI. Primary Voice5 was independently delivered during this
task; use the fresh baseline, not the older6 task's version list.

## Delivery complete

Exact anonymous raw-main channel refreshed to7/13 assets. Ordinary full Workspace
rebuild completed with37 apps/451 payloads; complete managed inventory audit and
every archived payload size/SHA256 verified. This is local114 QA, no suite release.
Normal Tube-only owner updates completed on primary in2m40s and secondary2m47s.
All nine stages pass, startup healthy, latest verified recovery kept. Each retired
one older automatic update recovery through normal cleanup; unrelated scheduled/
manual backups retained. Fresh checks on both show7 up to date and every unrelated
displayed installed version unchanged from the immediate pre-apply baseline.

Primary all four installed payload sizes/hashes equal7; transaction phase complete.
Both deployments serve new index-CTn-3iX-.js and display7. CSS/logo bytes bundled
through the supported browser asset capability equal the tested release. Both new
import dialogs show10/daily and all requested choices. Secondary existing source
still0/Unlimited with24-hour schedule; no owner import or settings were submitted
as a fixture. Completed receipts, version comparisons, payload evidence and final
dialog screenshot remain outside Git in the task artifact root. Updater tabs
closed; existing app views retained. No pending delivery target or required work.
No live YouTube download, Linux runtime, physical-phone or reboot claim is added.
