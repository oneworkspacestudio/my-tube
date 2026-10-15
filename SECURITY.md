# Security

Report vulnerabilities privately through the repository owner's GitHub profile
or GitHub private vulnerability reporting when available. Do not attach tokens,
session cookies, user databases, original media or private logs to public issues.

This is an unsigned preview, not an independently audited public hosting product.
Use HTTPS for non-loopback URLs and a dedicated service account with access only
to its own data/media folders. Keep FFmpeg and FFprobe current. Media decoders
process untrusted input: protocol allowlists, process memory/worker/thread/time
limits reduce risk but are not a full operating-system sandbox. Linux/macOS do
not currently have the Windows Job Object memory limit; those platforms are
development targets, not packaged acceptance targets.

Managed startup refuses filesystems without persistent ACLs on Windows. Directory
ACLs grant the process identity, SYSTEM and local Administrators. Windows admins
can read plaintext media, SQLite and backups. No end-to-end encryption is claimed.
The service must use the identity that can access its protected data and secret
environment. Do not serve these folders through a separate static file server.

Every metadata/original/poster/HLS/segment/subtitle route uses the same video
authorization predicate. Knowing a UUID never grants private access. Anonymous
public access is off by default. Unlisted videos require a direct link and obey
the server's anonymous-access setting. An application administrator can moderate
reported content but does not receive private playback permission.

Sessions use hashed opaque tokens with fixed expiration, HttpOnly/SameSite=Lax
cookies, Secure on HTTPS, and CSRF plus exact Origin on writes. The OIDC callback
binds one-use state to its initiating browser, with nonce and PKCE S256, and
validates issuer, audience, token signature and expiry through coreos/go-oidc.
My Tube logout revokes its own session only. First-run/local owner links are
single-use and loopback-only; do not record or share them.

Rate limits are per socket peer; reverse proxies share a bucket. Forwarded-IP
headers are deliberately not trusted. Put a deployment-aware limiter in front
of a large public instance. Analytics are client-reported and not fraud proof.
Upload quotas reserve declared original bytes, count retained deleted originals,
check configured physical capacity, and bound concurrent uploads. Processing
also monitors storage, but filesystem writes can exceed a limit briefly between
checks; keep operating-system disk reserves and quotas for production workloads.
