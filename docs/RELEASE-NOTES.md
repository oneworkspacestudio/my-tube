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
