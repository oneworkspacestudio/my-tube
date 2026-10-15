# REST API

Paths below are relative to `PUBLIC_URL` (including `/my-tube` when managed).
JSON responses use lower_snake_case. Errors return `{ "error": "message" }`.
All writes require a valid session, exact configured Origin and X-CSRF-Token
from GET /api/me. IDs supplied by a client never establish the current account.

| Routes | Methods / purpose |
| --- | --- |
| /healthz | GET: process/database health and version; no user content |
| /auth/login, /auth/callback | GET: discovery, code/PKCE and callback |
| /auth/local | GET: one-use loopback-only setup/local login |
| /api/me, /api/logout | GET profile/config capability flags; POST revoke session |
| /api/channel | POST create/update own channel: name, handle, description, visibility |
| /api/channels, /api/channels/{id} | GET channel directory/profile |
| /api/channel/image/{avatar\|banner} | POST owned image bytes (JPEG/PNG) |
| /api/channels/{id}/image/{avatar\|banner} | GET authorized channel image |
| /api/videos | GET discovery/search/filter/owner/history/feed lists |
| /api/videos/{id} | GET metadata; PUT owned metadata; DELETE recoverably retire |
| /api/uploads | GET own pending uploads; POST name,title,size,visibility |
| /api/uploads/{id} | GET checkpoint; PATCH chunk bytes + Upload-Offset; DELETE cancel |
| /api/uploads/{id}/complete | POST validate, hash, publish and queue processing |
| /api/videos/{id}/media/{file} | GET/HEAD authorized original, poster.jpg, captions.vtt, master.m3u8, rendition and segment files |
| /api/videos/{id}/thumbnail, /subtitles | POST image / WebVTT bytes |
| /api/videos/{id}/access | PUT {subject,allow}: exact issuer subject already provisioned |
| /api/videos/{id}/retry | POST retry processing |
| /api/videos/{id}/replace | POST {replacement_id}: own verified private upload; new URL |
| /api/videos/{id}/like, /later | POST {enabled} |
| /api/videos/{id}/progress | POST {position,seconds}; seconds capped by elapsed time |
| /api/history | DELETE own history |
| /api/channels/{id}/subscribe | POST {enabled} |
| /api/videos/{id}/comments | GET visible comments; POST {body,parent} |
| /api/comments/{id} | PUT own {body}; DELETE by author/video owner |
| /api/playlists | GET own playlists or visible owner playlists; POST title,description,visibility |
| /api/playlists/{id} | GET, PUT, DELETE; writes require ownership |
| /api/playlists/{id}/videos | POST {video_id}; DELETE /{video_id} removes item |
| /api/notifications, /api/notifications/read | GET own notifications; POST mark read |
| /api/preferences | PUT {personalize} |
| /api/studio | GET own aggregate analytics |
| /api/videos/{id}/report | POST {reason} |
| /api/drive/list, /api/drive/import | POST {source,path,size?}; optional X-Drive-Token |
| /api/videos/{id}/drive-export | POST {source,path}; only own originals |
| /api/admin | GET config, users, job status, reports, storage and audit |
| /api/admin/config | PUT full validated config; requires restart |
| /api/admin/users/{id} | PUT {disabled,quota_mb} |
| /api/admin/reports/{id} | PUT {status,remove} |
| /api/admin/reconcile | POST {purge_deleted:false}; explicit true permanently purges deleted media |

Video list parameters: q, category, channel, days, duration=short/long,
sort=recent/popular/oldest, playlist, view=studio/subscriptions/history/later/likes/shorts,
recommend=1. Search quotes user text as an FTS phrase, never as SQL. Results
are bounded to 500 candidates. Shorts use vertical videos up to 180 seconds.
Playlist search supports q. Direct unlisted access is separate from discovery.

Chunks have a maximum of 8 MiB. Offset conflicts return HTTP 409 and the current
Upload-Offset header; fetch status before retrying. Finalization is idempotent.
Container/probe/duplicate failures retain the pending upload for explicit retry
or cancellation. The original endpoint supports Range and ?download=1. Cookies
authorize HLS requests; segment URLs remain relative to the same protected route.
