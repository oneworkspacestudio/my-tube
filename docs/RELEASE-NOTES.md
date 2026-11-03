# v0.1.0-preview.6

My Tube now uses One Workspace's blue colour family in light and dark themes,
with its original repository logo in the header and browser tab. Navigation,
buttons, default avatars and placeholder artwork share the new palette. The
One Workspace label stays visible on phones, and narrow screens no longer gain
a horizontal scrollbar from the category strip.

Existing accounts, media, imported sources, settings and permissions stay in
place. There is no database or managed application contract change. Use the
normal backed-up Workspace app updater. Windows x64 and Linux x64 builds are
provided; this remains an unsigned preview. Build and focused visual acceptance
are recorded in build-info.json and the compact repository history. Linux runtime,
clean-host/service/reboot and broad playback/import acceptance are not claimed.

# v0.1.0-preview.1

First usable Windows preview of My Tube for One Workspace.

Includes MyTube.exe, embedded React UI and SQLite, resumable uploads, direct
MP4 and optional HLS processing, channels/Creator Studio, search and social
features, MyAccounts PKCE, scoped Quantum transfers, portable recovery and a
digest-pinned Workspace app manifest. No Docker, Node or Python runtime needed.
FFmpeg/FFprobe are separate optional helpers and are not redistributed.

Tests include real MyAccounts/Quantum/helper fixtures, native restart/restore and
Chrome desktop/mobile workflows. See ACCEPTANCE.md for exact evidence and the
remaining clean-machine, service/reboot and live Workspace deployment gates.
The executable is unsigned. This release is a prerelease, not a stable build.
