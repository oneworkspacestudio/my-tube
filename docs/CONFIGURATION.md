# Configuration

See the complete annotated-by-name `config.example.json`. Paths are relative to
the configuration file; CLI overrides may be absolute. Arguments precede the
positional command. Changes saved in administration take effect on restart.
Invalid configuration fails before serving requests.

| Setting | Default / behavior |
| --- | --- |
| listen / public_url | 127.0.0.1:8796; include the proxy route in public_url |
| data_dir / media_dir | data / data/media; dedicated folders, managed NTFS |
| mode | managed; explicit local mode is loopback-only |
| oidc_issuer / oidc_client_id | from configuration or Workspace environment |
| admin_subjects | explicit exact MyAccounts subjects; no first-user promotion |
| session_hours | 8; fixed 1–24 hours |
| upload_mb / quota_mb / storage_mb | 2048 / 10240 / 102400 MiB |
| allowed_formats | .mp4, .mov, .mkv, .webm; actual content is verified |
| public_uploads / public_access | both false; separate publish/anonymous controls |
| workers / threads / job_minutes | 1 / 2 / 120; max 4 workers and 8 threads |
| transcode / qualities | true; [360,480,720,1080], bounded rates and no upscaling; explicit false is respected |
| auto_helpers | true on Windows x64; one-time verified115MB upstream download into private data/helpers; false disables downloads |
| processing_hours_utc | [] for all hours; e.g. [0,1,2,3,4] for overnight processing |
| ffmpeg / ffprobe | explicit executable paths or adjacent helpers/ executables |
| scan_program | optional executable receiving the staged file as its sole argument; nonzero/timeout rejects publication |
| detailed_analytics / retention_days | false / 90; unique account counts shown only when enabled |
| drive_url | optional configured Quantum base URL, HTTPS except loopback |

MiB values are binary units. Quotas count retained deleted originals until an
administrator explicitly purges them. Partial uploads reserve declared sizes.
Thumbnails, derived streams and physical incoming files count toward storage
capacity. Storage monitoring is bounded polling; retain free disk headroom.

Environment overrides: MYTUBE_LISTEN, MYTUBE_DATA_DIR, MYTUBE_MEDIA_DIR,
MYTUBE_PUBLIC_URL, MYTUBE_DRIVE_URL, MYTUBE_OIDC_ISSUER,
MYTUBE_OIDC_CLIENT_ID, MYTUBE_OIDC_CLIENT_SECRET, MYTUBE_ADMIN_SUBJECTS.
WORKSPACE_OIDC_ISSUER/CLIENT_ID/CLIENT_SECRET/REDIRECT_URI are also recognized.
The callback can establish the external public prefix without a config file.
No secret is returned through the admin configuration API or written to its JSON.

Use HTTPS away from loopback. Managed HTTP localhost is supported for a local
Workspace or disposable development fixture. Do not set the issuer to a token
endpoint; use the exact issuer advertised by MyAccounts discovery.
