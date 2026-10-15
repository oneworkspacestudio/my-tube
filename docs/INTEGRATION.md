# One Workspace and My Drive

The app manifest follows the current
[ADDING-APPS contract](https://github.com/oneworkspacestudio/one-workspace/blob/main/docs/ADDING-APPS.md).
`scripts/package.ps1` reads the actual executable version, byte size and SHA256.
The launcher registers native OIDC, protects its generated client secret and
injects exact callback environment values. The app uses relative assets and a
server-injected base path for both prefix-stripping and prefix-preserving proxies.
All My Tube API URLs are under `/my-tube/api`, never the Drive-owned root `/api`.

Stop the suite, back up its configuration, and follow its documented `add-app`
procedure with the release executable and `my-tube.workspace.json`. ID is
`my-tube`, name `My Tube`, route `/my-tube`, port 8796, direct health `/healthz`.
Use an available port if that one is already assigned. The manifest defaults
Drive to the suite origin and data to `${DATA}/my-tube`. Configure explicit
My Tube admin subjects through a protected config or environment. Local OIDC
clients may use localhost; non-loopback origins must use HTTPS. No duplicate
wizard appears when the launcher supplies an issuer and client.

Public video settings cannot bypass the launcher's Workspace session boundary.
If you expose an independent My Tube origin through Cloudflare, configure
public_url and the exact IdP redirect accordingly, retain HTTPS and auth checks,
and configure tunnel size/time limits for chunk uploads. My Tube never changes
your tunnel or live suite installation by itself.

## Quantum adapter

The implementation was checked against `oneworkspacestudio/drive`:

- `backend/swagger/docs/swagger.json`
- `backend/internal/web/auth.go` (Bearer token parsing)
- `backend/internal/web/resource.go` (scoped resource operations)
- `frontend/src/api/resources.js`

It calls `GET /api/users?username=self`, `GET /api/resources?source=...&path=...`,
`GET /api/resources/download?source=...&file=...`, and
`POST /api/resources?source=...&path=...&override=false&isDir=false` with a raw body.
These are actual Quantum endpoints; permissions are enforced by Quantum.

On the suite’s shared origin, the user's `filebrowser_quantum_jwt` cookie can
be forwarded server-side to the configured Drive origin. On a separate origin,
the user can enter their own scoped API token in the import/export dialog. It
is kept only in that dialog’s memory, never in localStorage/config/database.
Redirects are rejected so credentials cannot move to another origin.
Tokens with effective administrator permission are rejected. If your account
is an administrator, create an appropriately limited named token for this use.

Select a source name and a scoped path. Browse lists clickable current items;
select a folder then browse again, or select a video and confirm Import.
Import is explicit and private by default; source bytes remain untouched.
Export creates a new file and refuses overwrite. Unknown content length requires
the user to supply the file’s exact size; quotas and byte counts are checked.
No undocumented filesystem shortcut or shared administrator token is used.
