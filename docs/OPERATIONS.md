# Operations and recovery

## Commands

Flags precede commands. `--config` defaults beside the EXE. `--data`, `--media`,
`--port`, `--public-url`, `--no-browser`, `--local` and `--file` are supported.

```powershell
MyTube.exe version
MyTube.exe config-check
MyTube.exe health
MyTube.exe migrate
MyTube.exe --no-browser run
# Elevated shell, configured managed mode only:
MyTube.exe --config C:\MyTube\config.json install
MyTube.exe start
MyTube.exe stop
MyTube.exe uninstall
```

The service is `OneWorkspaceMyTube`. Configure the service identity’s access to
the data/media paths and its OIDC secret environment. CLI data/media/config
paths are retained in service arguments; port/public settings belong in config.
Service installation and a real machine reboot are not yet acceptance-tested.
The instance lock refuses a second app or offline operation using the same data.

## Backups

Stop the app or service, then run `MyTube.exe --file D:\Backups\tube.zip backup`.
The output must not already exist. Backups include a consistent SQLite snapshot,
originals, acknowledged upload staging, HLS, posters and captions. Each file has
a SHA256 in a manifest. No secret environment/config is included. Original
relationships and sizes are checked before backup. Incomplete/missing media
must be reconciled or recovered before a successful backup is reported.

Keep several generations on a different disk. Backups are plaintext. Protect
them like the originals and test restoration regularly. Keep a separate protected
copy of your non-secret configuration and OIDC client registration/secret source.

Restore into a fresh dedicated `--data` folder with its default `media/` location.
The restorer rejects unsafe/duplicate archive paths, unexpected files, excessive
size, bad hashes, inconsistent SQLite data and missing original relationships.
Sessions and pending OIDC logins are invalidated. It refuses an existing database.
Restart using the restored folder; relocate the media while stopped if needed.

## Uploads and processing

The browser acknowledges only chunks that were synced and checkpointed. After a
browser restart, reopen Upload and select the same file by name and size to
resume. Following a forced app exit, startup trims any unacknowledged tail and
returns interrupted finalization to staging. It never invents acknowledged bytes.
Staging uploads can be cancelled explicitly; they are not silently aged out.

Running jobs return to queued after restart. Configure helpers then choose
Process to retry a failed job. A failure retains the original. HLS output is
published atomically after completion; stale temporary work is rebuilt. Keep
headroom for the original and all configured quality tiers.

Delete stops access immediately but retains the original for recovery. Replacing
a video atomically retires the old record and publishes a verified new private
upload with copied metadata and a **new URL**; old social activity is not moved.
The previous original remains retained. Admin storage reconciliation reports
missing originals and removes orphan posters. `POST /api/admin/reconcile` with
`purge_deleted:true` permanently removes explicitly deleted media, skipping
running jobs. Back up before using this irreversible maintenance option.

Audit events live in SQLite and structured operational errors go to stderr.
History, notifications and audit rows are retained for retention_days at startup.
Public-instance spam controls include a request budget, comment/report limits,
per-user quotas and report moderation. Use a properly configured edge limiter
for untrusted high-volume public deployments.
