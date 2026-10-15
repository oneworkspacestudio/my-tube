# CODEX MASTER BUILD PROMPT
## My Tube — One Workspace YouTube Alternative

### 1. Mission

Build **My Tube**, a complete, open-source, self-hosted video-sharing platform for the One Workspace ecosystem.

My Tube should offer a polished, YouTube-inspired experience with video uploads, channels, subscriptions, recommendations, playlists, comments, search and streaming.

The platform must be suitable for personal servers, families and small communities, with a future architecture that could support larger public deployments.

**Critical requirements:**

- Windows-first deployment
- One main executable: `MyTube.exe`
- No Docker
- No external database server
- No Node.js or Python required at runtime
- MyAccounts OpenID Connect authentication
- Optional FileBrowser Quantum integration
- Fully functional browser-based interface
- Persistent data stored in accessible folders
- Low-resource operation
- Portable installation
- Open-source codebase

The outcome must be a working application, not a UI prototype.

### 2. Technology

Use the following preferred stack.

**Backend**
- Go
- SQLite with WAL mode
- REST API
- Background job processing
- Embedded frontend assets using Go embed
- HTTP range-request support
- Structured logging

**Frontend**
- React
- TypeScript
- Vite
- Tailwind CSS
- Lucide icons
- Responsive components
- TanStack Query or similar for data fetching

**Video processing**
- FFmpeg and FFprobe
- HTML5 video playback
- HLS adaptive streaming
- Thumbnail generation
- Poster extraction
- Video metadata analysis

**Authentication**
- MyAccounts via OIDC
- Secure session cookies
- Per-user authorisation

Use established libraries where practical.

Do not introduce heavyweight infrastructure without demonstrating why it is necessary.

### 3. Design and user experience

Create a highly polished YouTube-inspired interface with original One Workspace branding.

Use familiar YouTube navigation patterns but do not copy Google's logos, trademarks or proprietary assets.

**Desktop homepage**

- Top navigation bar
- My Tube logo
- Global video search
- Upload button
- Notifications
- User avatar
- Collapsible left navigation
- Category filter chips
- Responsive video thumbnail grid

**Left navigation**

- Home
- Shorts
- Subscriptions
- History
- Watch Later
- Liked Videos
- Your Videos
- Your Channel
- Playlists
- Explore
- Settings

**Video cards**

Each card should show:

- Thumbnail
- Duration
- Video title
- Channel avatar
- Channel name
- View count
- Upload date
- Watched progress
- Context menu

**Video watch page**

Implement:

- Large video player
- Cinema mode
- Full-screen mode
- Playback quality selection
- Playback speed
- Subtitle selection
- Autoplay
- Like button
- Subscribe button
- Share button
- Save to playlist
- Description
- Chapter navigation
- Comments
- Recommended videos

**Mobile design**

Create a mobile-first responsive version with bottom navigation, touch-friendly playback controls, and appropriately sized upload and editing interfaces.

Provide light and dark themes.

### 4. MyAccounts integration

MyAccounts is the existing identity provider shared across One Workspace applications.

Implement OIDC using Authorization Code with PKCE.

Requirements:

- Discover OIDC configuration through the issuer metadata
- Validate ID tokens, issuer, audience, signature and expiry
- Map accounts using issuer and subject
- Use MyAccounts profile names and avatars where available
- Maintain secure application sessions
- Support logout and session expiration
- Support multiple users
- Protect every write operation with authorisation checks

Users should not create separate passwords for My Tube.

**First-time login:**

1. User opens My Tube.
2. User authenticates through MyAccounts.
3. My Tube provisions a local application profile.
4. User can optionally create a channel.
5. Existing account holders keep their channel, subscriptions and history.

Never rely on client-provided user IDs for access control.

Use the actual MyAccounts API and documentation rather than inventing endpoints.

### 5. Channels and creators

Every authenticated user should be able to create and manage a channel.

Implement:

- Channel name and handle
- Description
- Avatar and banner
- Channel URL
- Channel homepage
- Videos tab
- Playlists tab
- About tab
- Subscriber count
- Visibility settings
- Multiple channel support as a later extension

Create a **Creator Studio** where users can:

- Upload videos
- Edit titles and descriptions
- Set thumbnails
- Organise playlists
- Choose visibility
- Review comments
- View basic analytics
- Delete or replace videos
- Inspect transcoding status

### 6. Video uploading

Implement reliable uploads with:

- Drag-and-drop upload
- File picker
- Multiple file selection
- Chunked, resumable uploads
- Upload progress
- Retry support
- File size validation
- MIME and container validation
- Duplicate detection
- Interrupted upload recovery
- Server-side ownership checks

Supported initial formats should include MP4, MOV, MKV and WebM where FFmpeg supports decoding them.

Do not assume every uploaded codec can be played directly in a browser.

Store original files separately from generated streaming renditions.

Use temporary upload files and atomic finalisation to prevent incomplete videos from appearing as finished uploads.

### 7. Video processing and streaming

Use FFprobe to inspect uploaded media.

Use FFmpeg to generate browser-compatible streaming outputs.

Support:

- Direct playback where the original is compatible
- Byte-range streaming for compatible MP4
- HLS for transcoded playback
- Multiple quality renditions
- Automatic playback quality selection
- Audio transcoding when necessary
- Thumbnail extraction
- Poster generation
- Subtitle handling
- Video duration and resolution detection

Recommended initial quality tiers:

- 360p
- 480p
- 720p
- 1080p

Only generate renditions up to the source resolution and respect administrator resource limits.

Do not upscale low-resolution content unnecessarily.

**Important:**

Transcoding is CPU-intensive. Implement a configurable worker queue with bounded concurrency.

For lower-powered servers, allow the administrator to disable automatic transcoding, favour direct playback and optionally schedule processing during idle periods.

When FFmpeg is missing, provide clear setup instructions and keep existing compatible videos playable.

### 8. Single-executable packaging

Produce:

`MyTube.exe`

The executable must contain:

- Go HTTP server
- REST API
- React production assets
- Authentication integration
- SQLite migrations
- Background job manager
- Admin dashboard
- Media-library management

FFmpeg may be included as an optional managed helper executable or bundled with explicit extraction and licence compliance.

**Do not claim the product is literally one runtime binary if FFmpeg must run as a separate process.** The goal is a one-click main application with no manually installed runtime dependencies.

Example installation:

    OneWorkspace/
      Apps/
        MyTube/
          MyTube.exe
          config.json
          data/
            mytube.db
            cache/
            thumbnails/
            uploads/
            transcodes/
            logs/
          media/
            originals/
            streaming/

Allow administrators to select media storage on another disk.

Paths must be configurable and portable where possible.

The executable must support command-line options for:

- Start server
- Install Windows service
- Remove Windows service
- Check configuration
- Run database migrations
- Display version
- Perform health checks

### 9. FileBrowser Quantum integration

FileBrowser Quantum is already installed as **My Drive**.

My Tube must support two storage modes.

**Mode A — Standalone**

The user chooses a local media directory.

My Tube handles uploads, storage, playback and metadata without FileBrowser Quantum.

**Mode B — One Workspace integration**

My Tube connects to existing storage managed through FileBrowser Quantum.

Requirements:

- Use documented Quantum APIs when appropriate
- Reuse approved file locations
- Do not duplicate originals unnecessarily
- Allow importing videos from My Drive
- Allow exporting uploaded videos to My Drive
- Support secure file selection
- Maintain user permissions
- Store generated thumbnails and HLS renditions separately
- Avoid modifying original media unexpectedly

For local installations, direct filesystem access to explicitly configured shared directories may be more efficient than transferring large video files through the Quantum API.

Implement a storage adapter layer so the rest of My Tube does not depend on the storage provider.

Do not silently bypass FileBrowser Quantum user permissions through an administrator token.

### 10. Video visibility and permissions

Implement three video visibility levels.

**Public**
- Viewable by anyone if the administrator enables public access
- Can appear in search, recommendations and channel pages

**Unlisted**
- Accessible through a direct link
- Does not appear in public discovery

**Private**
- Accessible only to its owner and specifically authorised users

Add server-level controls allowing administrators to disable public uploads or public video access entirely.

Enforce visibility on playback, thumbnails, downloads, subtitles, metadata and derived HLS files.

Use signed or authorised playback URLs where needed.

### 11. Subscriptions and social features

Implement:

- Subscribe and unsubscribe
- Channel subscription feed
- Like and unlike videos
- Comments and replies
- Edit and delete own comments
- Comment moderation
- Watch Later
- Playlists
- Watch history
- Resume playback
- Notifications for new uploads

Initial notifications can be implemented within the application.

Email and push notifications can be added later as optional integrations.

Provide spam controls, rate limits, reporting and administrative moderation for public instances.

### 12. Search and recommendations

Implement full-text search using SQLite FTS5.

Search across:

- Video titles
- Descriptions
- Channel names
- Tags
- Playlist titles

Provide filters for:

- Upload date
- Duration
- Channel
- Popularity
- Sort order

Start with explainable recommendation logic rather than heavyweight AI.

Signals may include:

- Recently uploaded videos
- User subscriptions
- Previously watched categories
- Video tags
- Similar channels
- Popularity
- Watch completion

Provide an option to disable personalisation.

Avoid invasive tracking or collecting unnecessary personal information.

Recommendations must respect video visibility and permissions.

### 13. Creator analytics

Create a creator dashboard displaying:

- Total views
- Unique viewers where reliably measurable
- Watch time
- Subscriber growth
- Top videos
- Recent uploads
- Engagement counts

Avoid presenting estimated metrics as exact measurements.

Use lightweight aggregate tables and retention policies.

Provide an administrator option to disable detailed analytics.

### 14. Administration

Build an administration interface supporting:

- Server configuration
- Port selection
- Storage paths
- User and channel management
- Upload size limits
- Maximum storage quotas
- Allowed file formats
- Transcoding worker limits
- Available streaming qualities
- Public access settings
- Moderation reports
- Disk usage
- Background job status
- Logs
- Backup and restore
- Updates and version information

Include a first-run setup wizard for standalone installation.

When managed by One Workspace, read configuration from the central installer and avoid duplicate setup steps.

### 15. Security

Implement:

- OIDC validation
- Secure sessions
- CSRF protection
- Server-side authorisation
- Path traversal prevention
- File type verification
- Malware scanning integration where available
- Upload rate limiting
- Disk quota enforcement
- Safe media-processing subprocess execution
- Process timeouts
- Resource limits
- Content Security Policy
- XSS protections
- SQL injection prevention
- Secure HTTP headers
- Audit logging
- Private-by-default administration endpoints

Never build shell commands by concatenating untrusted filenames.

Prevent unauthorised access to raw video files and transcoded outputs.

Use least-privilege filesystem access.

### 16. Reliability and data safety

Implement:

- SQLite migrations
- Database backups
- Configurable backup location
- Upload recovery
- Graceful shutdown
- Restartable transcoding jobs
- Atomic file operations
- Missing-file reconciliation
- Orphaned-thumbnail cleanup
- Storage health checks
- Clear recovery instructions

Backups must preserve the relationship between database records and physical media files.

Do not delete uploaded originals merely because a transcoding job fails.

### 17. One Workspace integration

My Tube must integrate with the One Workspace installer and launcher.

Support:

- App registration
- Installation detection
- Start and stop
- Health endpoint
- Version reporting
- Update checks
- Configuration through the central installer
- MyAccounts discovery/configuration
- FileBrowser Quantum discovery/configuration
- Optional Cloudflare Tunnel integration

Allow My Tube to operate independently of other optional applications.

Avoid duplicate instances by using application-level instance locking and configurable port checks.

### 18. Development phases

**Phase 1 — Foundation**
- Create repository
- Establish Go API and React frontend
- Implement SQLite
- Create executable packaging
- Set up automated tests

**Phase 2 — Authentication**
- MyAccounts OIDC
- User profiles
- Secure sessions
- Account permissions

**Phase 3 — Video library**
- Upload files
- Store originals
- Extract metadata
- Generate thumbnails
- Build video list
- Implement video playback

**Phase 4 — YouTube-style UI**
- Homepage
- Sidebar
- Video grid
- Watch page
- Search
- Responsive mobile layout

**Phase 5 — Channels and creator tools**
- Channels
- Channel profiles
- Video management
- Playlists
- Creator Studio

**Phase 6 — Streaming**
- FFmpeg integration
- Transcoding queue
- HLS renditions
- Playback quality switching
- Subtitle support

**Phase 7 — Social features**
- Subscriptions
- Likes
- Comments
- Watch Later
- History
- Recommendations

**Phase 8 — One Workspace integration**
- FileBrowser Quantum
- Central launcher
- Installation discovery
- Shared authentication
- Optional remote access

**Phase 9 — Administration**
- User management
- Storage management
- Moderation
- Analytics
- Backup and restore

**Phase 10 — Production release**
- Build Windows executable
- Test on fresh Windows installation
- Test without Docker
- Test restart and recovery
- Test multi-user security
- Create GitHub release
- Publish installation documentation

At the end of each phase, leave the application in a working, testable state.

### 19. Testing requirements

Create automated tests for:

- First-run installation
- Authentication and logout
- Account separation
- Video uploads
- Interrupted uploads
- Playback permissions
- Direct MP4 streaming
- HLS streaming
- Transcoding failures
- Video deletion
- Thumbnail generation
- Search
- Channel permissions
- Subscriptions
- Comments
- Database migrations
- Backup restoration
- FileBrowser Quantum integration
- Windows restart behaviour

Use Playwright for browser testing where appropriate.

Test both authenticated and unauthenticated access, especially for private videos.

### 20. GitHub repository

Create a repository named:

`my-tube`

Use the existing GitHub connection where available and authorised.

Include:

- Source code
- README
- Screenshots
- Architecture documentation
- API documentation
- Configuration examples
- Licence information
- Development instructions
- Release workflow
- Windows executable

Do not claim the repository or release has been published unless the operation actually succeeds.

### 21. Release requirements

Produce an installable, usable release.

Minimum release deliverables:

- `MyTube.exe`
- README
- Example configuration
- Setup instructions
- Sample environment configuration
- Windows startup instructions
- Backup instructions
- Release notes
- Licence and third-party notices

Create a GitHub Actions workflow that builds and tests the Windows executable.

Pin dependencies and use reproducible builds where practical.

### 22. Final acceptance criteria

The project is considered ready for its first stable release when:

1. Users can launch My Tube on Windows without installing Docker.
2. A working web interface opens in their browser.
3. Users can authenticate through MyAccounts.
4. Users can create channels and upload videos.
5. Uploaded videos appear on channel pages.
6. Browser-compatible video playback works.
7. Videos can be searched, liked and added to playlists.
8. Subscriptions and comments persist correctly.
9. Different users have properly isolated private data.
10. Media and metadata survive server restarts.
11. FileBrowser Quantum integration works when configured.
12. Standalone storage works without Quantum.
13. The application runs from a portable directory.
14. The installer can register it with One Workspace.
15. Security and recovery tests pass.

### 23. Instructions to Codex

Start by inspecting the existing One Workspace architecture and any available MyAccounts and FileBrowser Quantum documentation.

Do not invent authentication or storage API endpoints.

Implement the application in working phases, starting with a fully functional upload-and-playback workflow.

Prefer direct browser playback before introducing transcoding complexity.

Use FFmpeg as a managed optional dependency, and verify its licensing requirements for redistribution.

Avoid microservices, Redis, PostgreSQL, MongoDB and Docker unless an unavoidable technical requirement is demonstrated.

Keep the architecture modular without adding unnecessary runtime services.

Do not stop after generating a plan or static prototype.

Write functioning code, run tests, fix failures, build the Windows executable and produce a release-ready project.

**The end result must feel like a professional YouTube-style platform while remaining simple enough for someone to install through One Workspace with minimal configuration.**