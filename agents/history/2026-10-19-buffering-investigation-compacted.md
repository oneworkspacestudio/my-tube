# Playback buffering investigation

User asks why their YouTube buffers. Asked whether this means Workspace My Tube,
local/remote, or youtube.com; no reply at this handoff. Investigated My Tube as
the likely ecosystem reference. Diagnosis is conditional on that interpretation.
User's latest standing release preference is local Windows builds without GitHub
Actions; the older preview1 handoff's hosted-CI workflow is historical evidence,
not current authorization to use Actions.

Existing installed manager remains preview.38 and My Tube preview.1 at its normal
managed route. Workspace40/PDF3 publication did not upgrade the live machine.
Read only safe config fields, aggregate media metadata and log classifications;
owner data was not used as a functional test fixture. No playback requests,
configuration changes, restarts, helper execution, transcoding or publication.

Concrete finding: one ready, direct-play 1920x1080 MP4; approximately75.8MiB over
30.8seconds, averaging20.6Mbps. Media tree has only the original MP4 and no HLS
playlists/segments. Managed settings omit transcode, which defaults false in
internal/app/config.go. Thus no adaptive smaller qualities are generated and
the client uses the original. This plausibly causes remote buffering when the
home uplink/viewer connection cannot sustain about2.6MB/s plus overhead; remote
throughput and the exact playback stall have not been measured, so this is not
proof of the sole cause.

Source supports byte ranges through http.ServeContent; portal reverse proxy does
not explicitly strip Range. Client uses hls.js when renditions exist, otherwise
native original MP4. Adjacent managed helpers are absent; this shell can find
FFmpeg/FFprobe on PATH, which does not prove availability in the running manager.
Do not automatically run or redistribute the old development Anaconda helper.
Current reviewed helpers would need explicit configuration and packaging/license
provenance. Existing CRF23 transcode settings have no maxrate and master BANDWIDTH
uses a height formula rather than measured rendition bandwidth; future quality
work should use bounded bitrate tiers and correct manifest bandwidth.

Cloudflare error tail was historical (latest13:46Z vs current14:49Z), mostly client
cancellations and earlier origin-not-running events. It cannot establish a current
tunnel failure. Earlier GitHub installer uploads shared the home uplink, but ended
at14:28Z and were no longer active during this diagnosis. Do not blame them as the
current established cause or present the tunnel as inherently broken.

Fresh source clone is I:/Projects/my-tube, matching org main4dab3a4 at inspection.
Older history names H: as its then-authoritative checkout; no old checkout was
modified. This handoff is uncommitted local investigation only. If user requests
fixing, enable/provision compatible media processing through normal managed setup,
produce safe adaptive renditions without touching originals, improve quality/
processing guidance, and verify only with isolated synthetic accounts/media. No
new tests were added/run for this question. Any release needs current-task
publication authorization and coordinated Workspace pins.
