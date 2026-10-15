// @ts-nocheck -- Generated dynamic contracts from Tube.aui; edit the AUI source.
import {
  Alert,
  Badge,
  Button,
  Card,
  Checkbox,
  Divider,
  Grid,
  Heading,
  Icon,
  Image,
  Input,
  Link,
  Metric,
  Page,
  Row,
  Section,
  Select,
  Spacer,
  Stack,
  Text,
} from "@/components/ui";

export function Tube({
  data,
  onAction,
}: {
  data: any;
  onAction: (name: string) => void;
}) {
  return (
    <Page>
      <Row region="topbar">
        <Row region="brand">
          <Button
            intent="menu"
            variant="icon"
            title="Toggle navigation"
            icon="Menu"
          />
          <Button intent="home" variant="brand">
            <Icon name="Clapperboard" />
            <Text>My Tube</Text>
            <Badge>WORKSPACE</Badge>
          </Button>
        </Row>
        <Row region="searchbar">
          <Input
            field="search"
            value={data.app.search}
            placeholder="Search videos, creators and ideas"
            title="Search videos"
          />
          <Button
            intent="search"
            variant="search"
            title="Search"
            icon="Search"
          />
        </Row>
        <Row region="top-actions">
          <Button
            intent="theme"
            variant="icon"
            title="Toggle theme"
            icon="SunMoon"
          />
          <Button intent="upload" variant="create" icon="Plus">
            Create
          </Button>
          <Button
            intent="notifications"
            variant="icon"
            title="Notifications"
            icon="Bell"
          />
          {data.app.signedIn ? (
            <Button
              intent="settings"
              variant="avatar"
              title="Account and settings"
            >
              {data.app.initials}
            </Button>
          ) : (
            <Link href={data.app.loginURL} variant="primary">
              Sign in
            </Link>
          )}
        </Row>
      </Row>
      <Row region={data.app.shellRegion}>
        <Section region="sidebar">
          {data.app.nav.map((item, i) => (
            <Button
              key={i}
              intent="navigate"
              payload={item.key}
              variant={item.variant}
              icon={item.icon}
            >
              {item.label}
            </Button>
          ))}
          <Divider />
          <Text tone="eyebrow">YOUR SPACE</Text>
          {data.app.libraryNav.map((item, i) => (
            <Button
              key={i}
              intent="navigate"
              payload={item.key}
              variant={item.variant}
              icon={item.icon}
            >
              {item.label}
            </Button>
          ))}
          <Divider />
          <Text tone="eyebrow">MAKE IT YOURS</Text>
          <Button intent="channel" variant="nav" icon="CircleUserRound">
            Your channel
          </Button>
          <Button intent="studio" variant="nav" icon="ChartNoAxesCombined">
            Creator Studio
          </Button>
          <Button intent="playlists" variant="nav" icon="ListVideo">
            Playlists
          </Button>
          <Button intent="settings" variant="nav" icon="Settings2">
            Settings
          </Button>
          <Spacer />
          <Stack region="sidebar-footer">
            <Icon name="Layers3" />
            <Text>One Workspace</Text>
            <Text tone="muted">Your videos. Your community.</Text>
            <Text tone="small">{data.app.version}</Text>
          </Stack>
        </Section>
        <Section region="main">
          {data.app.error && <Alert tone="error">{data.app.error}</Alert>}
          {data.app.toast && (
            <Row region="toast">
              <Icon name="Check" />
              <Text>{data.app.toast}</Text>
              <Button
                intent="dismiss"
                variant="icon"
                title="Dismiss"
                icon="X"
              />
            </Row>
          )}
          {data.app.loading && <Text tone="muted">Loading your library…</Text>}
          {data.app.setupScreen && (
            <Card region="settings-card">
              <Text tone="eyebrow">WELCOME TO YOUR VIDEO WORKSPACE</Text>
              <Heading level={1}>Make My Tube yours</Heading>
              <Text tone="body">
                Choose your storage, connect MyAccounts and set the server
                limits. Your setup link works only once and only on this
                computer.
              </Text>
              <Button intent="setup" variant="primary" icon="Settings2">
                Set up My Tube
              </Button>
              <Text tone="small">
                One Workspace supplies these settings automatically when
                managing this app.
              </Text>
            </Card>
          )}
          {data.app.homeScreen && (
            <>
              <Row region="chips">
                {data.app.categories.map((item, i) => (
                  <Button
                    key={i}
                    intent="category"
                    payload={item.name}
                    variant={item.variant}
                  >
                    {item.name}
                  </Button>
                ))}
              </Row>
              <Row region="page-heading">
                <Stack>
                  <Text tone="eyebrow">A LITTLE CURIOSITY GOES A LONG WAY</Text>
                  <Heading level={1}>{data.app.heading}</Heading>
                  <Text tone="muted">{data.app.subheading}</Text>
                </Stack>
                <Button
                  intent="filters"
                  variant="outline"
                  icon="SlidersHorizontal"
                >
                  Filters
                </Button>
              </Row>
              {data.app.showFilters && (
                <Row region="filter-panel">
                  <Select
                    field="sort"
                    value={data.app.sort}
                    options={data.app.sortOptions}
                    title="Sort videos"
                  />
                  <Select
                    field="duration"
                    value={data.app.duration}
                    options={data.app.durationOptions}
                    title="Video duration"
                  />
                  <Select
                    field="days"
                    value={data.app.days}
                    options={data.app.dateOptions}
                    title="Upload date"
                  />
                </Row>
              )}
              {data.app.hasVideos ? (
                <Grid region="video-grid">
                  {data.app.videos.map((item, i) => (
                    <Card key={i} region="video-card">
                      <Button
                        intent="watch"
                        payload={item.id}
                        variant="thumbnail"
                        title={item.title}
                      >
                        <Image
                          src={item.poster}
                          alt={item.title}
                          fallback="video"
                        />
                        <Badge tone="duration">{item.durationLabel}</Badge>
                        {item.progress && (
                          <Metric value={item.progress} tone="progress" />
                        )}
                      </Button>
                      <Row region="video-info">
                        <Button
                          intent="visit-channel"
                          payload={item.channel_id}
                          variant="channel-avatar"
                        >
                          {item.initials}
                        </Button>
                        <Stack>
                          <Button
                            intent="watch"
                            payload={item.id}
                            variant="video-title"
                          >
                            {item.title}
                          </Button>
                          <Button
                            intent="visit-channel"
                            payload={item.channel_id}
                            variant="channel-name"
                          >
                            {item.channel}
                          </Button>
                          <Text tone="small">{item.stats}</Text>
                          <Text tone="reason">{item.recommendation}</Text>
                        </Stack>
                        <Button
                          intent="video-menu"
                          payload={item.id}
                          variant="icon"
                          title="Video options"
                          icon="EllipsisVertical"
                        />
                      </Row>
                    </Card>
                  ))}
                </Grid>
              ) : (
                <Stack region="empty-state">
                  <Icon name="Clapperboard" />
                  <Heading level={2}>{data.app.emptyTitle}</Heading>
                  <Text tone="muted">{data.app.emptyBody}</Text>
                  <Button intent="upload" variant="primary" icon="Upload">
                    Upload a video
                  </Button>
                </Stack>
              )}
            </>
          )}
          {data.app.watchScreen && (
            <Row region={data.app.watchRegion}>
              <Stack region="watch-main">
                <Section region="player">
                  <Input kind="player" video={data.app.video} />
                </Section>
                <Row region="player-controls">
                  <Button
                    intent="cinema"
                    variant="outline"
                    icon="RectangleHorizontal"
                  >
                    Cinema
                  </Button>
                  <Checkbox field="autoplay" checked={data.app.autoplay}>
                    Autoplay next
                  </Checkbox>
                  <Text tone="small">
                    Use the player for speed, captions and fullscreen
                  </Text>
                </Row>
                <Heading level={1}>{data.app.video.title}</Heading>
                <Row region="watch-actions">
                  <Button
                    intent="visit-channel"
                    payload={data.app.video.channel_id}
                    variant="channel-avatar"
                  >
                    {data.app.video.initials}
                  </Button>
                  <Stack>
                    <Button
                      intent="visit-channel"
                      payload={data.app.video.channel_id}
                      variant="video-title"
                    >
                      {data.app.video.channel}
                    </Button>
                    <Text tone="small">{data.app.video.stats}</Text>
                  </Stack>
                  <Button
                    intent={data.app.subscribeIntent}
                    payload={data.app.video.id}
                    variant={data.app.subscribeVariant}
                    icon="Bell"
                  >
                    {data.app.subscribeLabel}
                  </Button>
                  <Spacer />
                  <Button
                    intent="like"
                    variant={data.app.likeVariant}
                    icon="ThumbsUp"
                  >
                    {data.app.likeLabel}
                  </Button>
                  <Button intent="share" variant="pill" icon="Share2">
                    Share
                  </Button>
                  <Button intent="save" variant="pill" icon="ListPlus">
                    Save
                  </Button>
                  <Button
                    intent="video-menu"
                    payload={data.app.video.id}
                    variant="icon"
                    title="More video actions"
                    icon="Ellipsis"
                  />
                </Row>
                <Card region="description">
                  <Text tone="strong">{data.app.video.stats}</Text>
                  <Text tone="body">{data.app.video.description}</Text>
                  <Row region="tags">
                    <Text tone="small">{data.app.video.tags}</Text>
                  </Row>
                  {data.app.chapters.map((item, i) => (
                    <Button
                      key={i}
                      intent="seek"
                      payload={item.time}
                      variant="chapter"
                    >
                      {item.label}
                    </Button>
                  ))}
                </Card>
                <Row region="page-heading">
                  <Heading level={2}>{data.app.commentHeading}</Heading>
                  <Text tone="small">Keep the conversation kind.</Text>
                </Row>
                {data.app.signedIn && (
                  <Row region="comment-compose">
                    <Input
                      field="comment"
                      value={data.app.comment}
                      placeholder={data.app.commentPlaceholder}
                      title="Write a comment"
                      kind="textarea"
                    />
                    <Button intent="post-comment" variant="primary">
                      Post
                    </Button>
                  </Row>
                )}
                {data.app.comments.map((item, i) => (
                  <Card key={i} region={item.region}>
                    <Row>
                      <Badge tone="comment-avatar">{item.initials}</Badge>
                      <Text tone="strong">{item.name}</Text>
                      <Text tone="small">{item.date}</Text>
                    </Row>
                    <Text tone="body">{item.body}</Text>
                    <Row>
                      <Button intent="reply" payload={item.id} variant="text">
                        Reply
                      </Button>
                      {item.canEdit && (
                        <Button
                          intent="edit-comment"
                          payload={item.id}
                          variant="text"
                        >
                          Edit
                        </Button>
                      )}
                      {item.canDelete && (
                        <Button
                          intent="delete-comment"
                          payload={item.id}
                          variant="text"
                        >
                          Remove
                        </Button>
                      )}
                    </Row>
                  </Card>
                ))}
              </Stack>
              <Stack region="recommendations">
                <Heading level={2}>Keep exploring</Heading>
                {data.app.recommended.map((item, i) => (
                  <Card key={i} region="recommendation">
                    <Button
                      intent="watch"
                      payload={item.id}
                      variant="thumbnail"
                    >
                      <Image
                        src={item.poster}
                        alt={item.title}
                        fallback="video"
                      />
                      <Badge tone="duration">{item.durationLabel}</Badge>
                    </Button>
                    <Stack>
                      <Button
                        intent="watch"
                        payload={item.id}
                        variant="video-title"
                      >
                        {item.title}
                      </Button>
                      <Text tone="small">{item.channel}</Text>
                      <Text tone="small">{item.stats}</Text>
                      <Text tone="reason">{item.recommendation}</Text>
                    </Stack>
                  </Card>
                ))}
              </Stack>
            </Row>
          )}
          {data.app.studioScreen && (
            <>
              <Row region="page-heading">
                <Stack>
                  <Text tone="eyebrow">CREATOR STUDIO</Text>
                  <Heading level={1}>A space for your stories</Heading>
                  <Text tone="muted">
                    Upload, publish and see how your community is growing.
                  </Text>
                </Stack>
                <Button intent="upload" variant="primary" icon="Plus">
                  Upload video
                </Button>
              </Row>
              <Grid region="stats">
                <Card>
                  <Icon name="Video" />
                  <Text tone="muted">Videos</Text>
                  <Metric value={data.app.analytics.videos} />
                </Card>
                <Card>
                  <Icon name="Eye" />
                  <Text tone="muted">Views</Text>
                  <Metric value={data.app.analytics.views} />
                </Card>
                <Card>
                  <Icon name="Clock3" />
                  <Text tone="muted">Watch time</Text>
                  <Metric value={data.app.watchTime} />
                </Card>
                <Card>
                  <Icon name="UsersRound" />
                  <Text tone="muted">Subscribers</Text>
                  <Metric value={data.app.analytics.subscribers} />
                </Card>
              </Grid>
              <Text tone="small">{data.app.analytics.measurement}</Text>
              <Heading level={2}>Your content</Heading>
              {data.app.ownedVideos.map((item, i) => (
                <Row key={i} region="studio-video">
                  <Image src={item.poster} alt={item.title} fallback="video" />
                  <Stack>
                    <Button
                      intent="watch"
                      payload={item.id}
                      variant="video-title"
                    >
                      {item.title}
                    </Button>
                    <Text tone="small">{item.statusLabel}</Text>
                    <Text tone="small">{item.error}</Text>
                  </Stack>
                  <Spacer />
                  <Badge>{item.visibility}</Badge>
                  <Button
                    intent="retry"
                    payload={item.id}
                    variant="outline"
                    icon="RefreshCw"
                  >
                    Process
                  </Button>
                  <Button
                    intent="edit-video"
                    payload={item.id}
                    variant="outline"
                    icon="Pencil"
                  >
                    Edit
                  </Button>
                </Row>
              ))}
            </>
          )}
          {data.app.channelScreen && (
            <>
              <Section region="channel-banner">
                <Image
                  src={data.app.channelBanner}
                  alt="Channel banner"
                  fallback="banner"
                />
              </Section>
              <Row region="channel-heading">
                <Input
                  kind="channelAvatar"
                  src={data.app.channelAvatar}
                  text={data.app.channelInitials}
                />
                <Stack>
                  <Heading level={1}>{data.app.channelData.name}</Heading>
                  <Text tone="muted">{data.app.channelDetails}</Text>
                  <Text tone="body">{data.app.channelData.description}</Text>
                </Stack>
                <Spacer />
                {data.app.ownChannel && (
                  <Button intent="edit-channel" variant="outline" icon="Pencil">
                    Customize channel
                  </Button>
                )}
              </Row>
              <Row region="chips">
                <Button intent="channel-videos" variant="chip">
                  Videos
                </Button>
                <Button intent="channel-playlists" variant="chip">
                  Playlists
                </Button>
                <Button intent="channel-about" variant="chip">
                  About
                </Button>
              </Row>
              {data.app.channelAbout ? (
                <Card>
                  <Heading level={2}>About this channel</Heading>
                  <Text tone="body">{data.app.channelData.description}</Text>
                </Card>
              ) : (
                <Stack>
                  {data.app.channelPlaylists ? (
                    <Stack>
                      {data.app.channelLists.map((item, i) => (
                        <Button
                          key={i}
                          intent="open-playlist"
                          payload={item.id}
                          variant="playlist"
                          icon="ListVideo"
                        >
                          {item.title}
                        </Button>
                      ))}
                    </Stack>
                  ) : (
                    <Grid region="video-grid">
                      {data.app.channelVideos.map((item, i) => (
                        <Card key={i} region="video-card">
                          <Button
                            intent="watch"
                            payload={item.id}
                            variant="thumbnail"
                          >
                            <Image
                              src={item.poster}
                              alt={item.title}
                              fallback="video"
                            />
                            <Badge tone="duration">{item.durationLabel}</Badge>
                          </Button>
                          <Button
                            intent="watch"
                            payload={item.id}
                            variant="video-title"
                          >
                            {item.title}
                          </Button>
                          <Text tone="small">{item.stats}</Text>
                        </Card>
                      ))}
                    </Grid>
                  )}
                </Stack>
              )}
            </>
          )}
          {data.app.playlistsScreen && (
            <>
              <Row region="page-heading">
                <Stack>
                  <Text tone="eyebrow">MADE FOR ANOTHER MOMENT</Text>
                  <Heading level={1}>Your playlists</Heading>
                  <Text tone="muted">
                    Keep good videos together. Private until you decide to
                    share.
                  </Text>
                </Stack>
                <Button intent="new-playlist" variant="primary" icon="Plus">
                  New playlist
                </Button>
              </Row>
              <Grid region="playlist-grid">
                {data.app.playlists.map((item, i) => (
                  <Card key={i} region="playlist-card">
                    <Icon name="ListVideo" />
                    <Button
                      intent="open-playlist"
                      payload={item.id}
                      variant="video-title"
                    >
                      {item.title}
                    </Button>
                    <Text tone="small">{item.subtitle}</Text>
                    <Button
                      intent="edit-playlist"
                      payload={item.id}
                      variant="text"
                    >
                      Edit playlist
                    </Button>
                  </Card>
                ))}
              </Grid>
            </>
          )}
          {data.app.notificationsScreen && (
            <>
              <Heading level={1}>Your notifications</Heading>
              <Button intent="read-notifications" variant="outline">
                Mark all read
              </Button>
              {data.app.notifications.map((item, i) => (
                <Button
                  key={i}
                  intent="watch"
                  payload={item.video_id}
                  variant="notification"
                  icon="Bell"
                >
                  {item.body}
                </Button>
              ))}
              <Text tone="muted">
                New uploads from channels you follow appear here.
              </Text>
            </>
          )}
          {data.app.settingsScreen && (
            <>
              <Row region="page-heading">
                <Stack>
                  <Text tone="eyebrow">YOUR EXPERIENCE</Text>
                  <Heading level={1}>Settings</Heading>
                  <Text tone="muted">
                    A quieter, more personal video library.
                  </Text>
                </Stack>
              </Row>
              <Card region="settings-card">
                <Heading level={2}>Account & privacy</Heading>
                <Text>{data.app.accountLabel}</Text>
                <Checkbox field="personalize" checked={data.app.personalize}>
                  Personalise recommendations using subscriptions and watch
                  history
                </Checkbox>
                <Button intent="clear-history" variant="outline" icon="History">
                  Clear watch history
                </Button>
                <Button intent="theme" variant="outline" icon="SunMoon">
                  Switch light / dark theme
                </Button>
                <Button intent="logout" variant="outline" icon="LogOut">
                  Sign out of My Tube
                </Button>
              </Card>
              <Card region="settings-card">
                <Heading level={2}>Storage & processing</Heading>
                <Text tone="body">{data.app.helperStatus}</Text>
                <Text tone="small">
                  FFmpeg runs as an optional separate helper process. Originals
                  stay intact when a job fails.
                </Text>
                {data.app.driveEnabled && (
                  <Button intent="drive" variant="outline" icon="FolderOpen">
                    Import from My Drive
                  </Button>
                )}
                <Link href="https://github.com/oneworkspacestudio/my-tube/releases">
                  Releases & updates
                </Link>
              </Card>
              {data.app.isAdmin && (
                <Card region="settings-card">
                  <Heading level={2}>Administration</Heading>
                  <Text tone="muted">
                    Manage resource limits, storage, background jobs and
                    reports.
                  </Text>
                  <Button intent="admin" variant="primary" icon="ShieldCheck">
                    Open administration
                  </Button>
                </Card>
              )}
            </>
          )}
          {data.app.adminScreen && (
            <>
              <Row region="page-heading">
                <Stack>
                  <Text tone="eyebrow">SERVER ADMINISTRATION</Text>
                  <Heading level={1}>Your library, under control</Heading>
                  <Text tone="muted">{data.app.adminSummary}</Text>
                </Stack>
                <Button intent="reconcile" variant="outline" icon="RefreshCw">
                  Check storage
                </Button>
              </Row>
              <Card region="settings-card">
                <Heading level={2}>Configuration</Heading>
                <Text tone="muted">
                  Settings take effect after restart. Keep client secrets in the
                  environment.
                </Text>
                <Input
                  field="adminConfig"
                  kind="textarea"
                  value={data.app.adminConfig}
                  title="Server configuration JSON"
                  rows={16}
                />
                <Button intent="save-config" variant="primary">
                  Save for next restart
                </Button>
                <Text tone="small">
                  Backup / restore: stop the server, then MyTube.exe --file
                  backup.zip backup. Restore into a fresh --data directory.
                </Text>
              </Card>
              <Heading level={2}>Users</Heading>
              {data.app.adminUsers.map((item, i) => (
                <Row key={i} region="studio-video">
                  <Text tone="strong">{item.name}</Text>
                  <Text tone="small">{item.quotaLabel}</Text>
                  <Spacer />
                  <Button
                    intent="manage-user"
                    payload={item.id}
                    variant="outline"
                  >
                    Manage
                  </Button>
                </Row>
              ))}
              <Heading level={2}>Background jobs</Heading>
              {data.app.jobs.map((item, i) => (
                <Row key={i} region="studio-video">
                  <Text tone="small">{item.video_id}</Text>
                  <Badge>{item.status}</Badge>
                  <Text tone="small">{item.error}</Text>
                </Row>
              ))}
              <Heading level={2}>Moderation reports</Heading>
              {data.app.reports.map((item, i) => (
                <Row key={i} region="studio-video">
                  <Text>{item.reason}</Text>
                  <Badge>{item.status}</Badge>
                  <Button
                    intent="resolve-report"
                    payload={item.id}
                    variant="outline"
                  >
                    Resolve
                  </Button>
                  <Button
                    intent="remove-reported"
                    payload={item.id}
                    variant="outline"
                  >
                    Remove video
                  </Button>
                </Row>
              ))}
              <Heading level={2}>Audit log</Heading>
              {data.app.logs.map((item, i) => (
                <Text key={i} tone="small">
                  {item.label}
                </Text>
              ))}
            </>
          )}
          {data.app.uploadScreen && (
            <>
              <Row region="page-heading">
                <Stack>
                  <Text tone="eyebrow">SHARE SOMETHING WORTH WATCHING</Text>
                  <Heading level={1}>Upload your videos</Heading>
                  <Text tone="muted">{data.app.uploadHelp}</Text>
                </Stack>
                <Button intent="studio" variant="outline" icon="ArrowLeft">
                  Creator Studio
                </Button>
              </Row>
              <Card region="dropzone">
                <Icon name="CloudUpload" />
                <Heading level={2}>Drop your next story here</Heading>
                <Text tone="muted">
                  MP4, MOV, MKV or WebM · originals are always preserved
                </Text>
                <Input
                  kind="files"
                  field="files"
                  multiple={true}
                  accept=".mp4,.mov,.mkv,.webm"
                  title="Select videos"
                />
                <Select
                  field="uploadVisibility"
                  value={data.app.uploadVisibility}
                  options={data.app.visibilityOptions}
                  title="Upload visibility"
                />
              </Card>
              {data.app.uploadTasks.map((item, i) => (
                <Card key={i} region="upload-task">
                  <Row>
                    <Text tone="strong">{item.name}</Text>
                    <Spacer />
                    <Badge>{item.label}</Badge>
                  </Row>
                  <Metric tone="progress" value={item.percent} />
                  <Text tone="small">{item.error}</Text>
                  <Button
                    intent="resume-upload"
                    payload={item.id}
                    variant="outline"
                    icon="RefreshCw"
                  >
                    Select same file to resume / retry
                  </Button>
                  <Button
                    intent="cancel-upload"
                    payload={item.id}
                    variant="text"
                  >
                    Cancel
                  </Button>
                </Card>
              ))}
              {data.app.driveEnabled && (
                <Button intent="drive" variant="outline" icon="FolderOpen">
                  Or import from My Drive
                </Button>
              )}
            </>
          )}
        </Section>
        <Row region="bottom-nav">
          <Button intent="home" variant="nav" icon="House">
            Home
          </Button>
          <Button intent="navigate" payload="shorts" variant="nav" icon="Flame">
            Shorts
          </Button>
          <Button intent="upload" variant="nav" icon="CirclePlus">
            Create
          </Button>
          <Button
            intent="navigate"
            payload="subscriptions"
            variant="nav"
            icon="TvMinimalPlay"
          >
            Following
          </Button>
          <Button intent="studio" variant="nav" icon="CircleUserRound">
            You
          </Button>
        </Row>
      </Row>
      {data.app.dialog && <Input kind="dialog" />}
    </Page>
  );
}
