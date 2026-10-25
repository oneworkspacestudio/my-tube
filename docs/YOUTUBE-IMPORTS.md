# Import YouTube channels and playlists

Open **YouTube imports** in My Tube’s sidebar, or choose **Import from YouTube**
on the upload page. Select **Add channel or playlist**, paste a channel URL,
`@handle` or playlist URL, choose 1–50 videos (default 10) and start the import.

- Channels check their newest regular uploads. Shorts and live tabs are not
  included in the channel import. Playlists default to the last selected number
  of entries, for playlists where new videos are added at the bottom. Choose the
  **Start of playlist** window for newest-first playlists. This follows playlist
  position, not a promised upload-date sort. Reading the end of a large playlist
  can take longer; source scanning is limited to five minutes.
- Each source has its own private My Tube playlist. Videos belong to the signed
  in account and default to private. Original titles, descriptions, creator names
  and source links are retained. A private library channel is created on the
  first import if the account has no channel; other creators are not impersonated.
- My Tube downloads the media onto its configured `media_dir`, including when
  that folder is on another drive. Videos use normal quota reservations,
  container inspection, optional scanning, atomic publication and HLS processing.
  Imported videos can use the existing user-scoped **Export to My Drive** action.
- Importing is a server background job. Leaving or closing the browser does not
  stop it. Keep the My Tube server running. One download runs at a time across
  the server. Up to 25 sources can be saved per account.
- **Check for new videos** skips previously imported video IDs, including IDs
  saved through another source owned by the same account. File hashes also
  prevent duplicate originals. A new source can link the existing video into
  its playlist without copying its media.
- **Daily** checks are optional and off by default. They check the configured
  recent window, not the complete backlog, once per day while the server is
  running. A missed check runs when the server starts; existing files are kept.
  Change the window/schedule from a source’s **Settings**.
- **Cancel import** stops the active helper tree, releases the incomplete upload
  and keeps saved videos. **Check for new videos** retries unavailable/failed
  entries in the current window. Interrupted downloads restart from the beginning;
  they are not byte-resumed. Published videos are recovered across restarts.
- **Remove source** stops tracking that source and keeps videos and playlist.
  Deleting the associated playlist also removes its saved source; an active
  import must first be cancelled. Deleting a video does not delete it on YouTube.

## Progress and limitations

The page distinguishes tool preparation, source checking, downloading, verifying
and saving, completion and failure. Download bytes and total size are shown when
YouTube supplies the total; otherwise the progress bar is indeterminate. A video
is counted as saved only after verification and publication. Completed/cancelled
items do not keep an active progress bar. Each failed video has a result and
other available videos in the batch can continue.

Imports select H.264/AAC MP4 at up to 720p. With FFmpeg/FFprobe available, separate
audio/video streams are muxed directly into a bounded fragmented MP4 stream;
there are no full separate/merged staging copies. Without these helpers, the
best available single-file MP4 is used, which may be a lower resolution. Existing
automatic FFmpeg preparation is used on Windows; Linux operators can configure
their FFmpeg/FFprobe paths as described in [media setup](MEDIA.md).

Only publicly downloadable videos are supported. Private, member-only,
sign-in-only, age-restricted, unavailable and live/upcoming videos may fail and
are reported per item. My Tube does not import browser cookies or account
passwords. YouTube can change formats, impose rate limits or restrict this
server’s IP; an import cannot guarantee availability. A channel or playlist with
no accessible videos reports a source error. No YouTube API key is needed.

## Helper provenance and resource boundaries

On the first explicit import, My Tube fetches these pinned separate programs
from the official releases for Windows x64 or Linux x64:

- [yt-dlp 2026.10.21](https://github.com/yt-dlp/yt-dlp/releases/tag/2026.10.21)
  ([source and licenses](https://github.com/yt-dlp/yt-dlp/tree/2026.10.21)).
- [Deno 2.9.8](https://github.com/denoland/deno/releases/tag/v2.9.8)
  ([MIT license](https://github.com/denoland/deno/blob/v2.9.8/LICENSE.md)).

Exact archive/executable sizes and SHA256 values are compiled in
`internal/app/youtube_tools.go`. Helpers reside below `data_dir/helpers/`, are
verified before use, and are never self-updated from an unpinned “latest” URL.
The installed helper files use about 111 MiB on Windows / 131 MiB on Linux,
plus temporary extraction space. Downloads and per-command scratch use the app’s
data folder, not the host system temp folder. Unknown cache files or links are
preserved and reported rather than recursively erased. All helper downloads use
HTTPS with bounded sizes, duration and redirect hosts.

Downloader configurations, plugins and remote component fetching are disabled.
Only normalized YouTube source URLs and validated video IDs enter process
arguments; no shell is used. Each video is time/byte/quota bounded. Windows uses
a kill-on-close job with process/memory limits; Unix uses a process group for
tree cancellation. This is not an operating-system sandbox for hostile decoders.

Source/item state uses additive SQLite schema version 2 and is included in normal
My Tube backups. Restore with this or a newer build; older schema-1 executables
cannot open the migrated database. No credentials are kept in source records.
