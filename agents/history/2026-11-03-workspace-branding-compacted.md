# My Tube One Workspace branding (2026-11-03)

## Scope and state

The user requested One Workspace design-system colours/branding and My Tube's
existing repository logo. Started from clean `main` at `86694e2`, equal to
`origin/main`. This task authorizes implementation; no commit, push, release,
catalogue promotion or installed-machine update was requested for this app.
The changes remain local and uncommitted. Existing preview5 assets are unchanged.

## Implementation

- Followed the suite blue family (`#2563EB`, `#60A5FA`, `#0B2A63`) from
  [suite branding guidance](https://github.com/oneworkspacestudio/one-workspace/blob/c2350a5/docs/BRANDING.md).
  Dark surfaces/text follow `internal/suite/navigation/appearance.css` from the
  same suite revision. The sibling checkout itself was not edited.
- Updated light/dark CSS tokens, selected navigation/categories, controls,
  default avatars, video placeholders, channel fallback banner and upload area.
  Semantic errors and creator artwork retain their meaning/content.
- Header and favicon use the unchanged `docs/branding/my-tube.svg` directly.
  Vite owns asset URLs and includes the artwork in the embedded frontend.
  Replaced the generic header clapperboard through `Tube.aui`, regenerated
  `Tube.generated.tsx`, and exposed the imported logo through the view model.
  The semantic Image adapter eagerly loads the header logo.
- The One Workspace label remains visible in the compact header. Fixed an
  existing 4px phone overflow: the category strip's negative margin now matches
  the 12px main padding at widths up to 400px.

## Verification

- Frontend generation, TypeScript and Vite production build passed. Vite still
  reports the existing large JS chunk advisory; dependencies are unchanged.
- Windows Go 1.27.3 embedded binary build passed. Review binary SHA256:
  `8892890291a9a9bb079395a9b5807a2e75e12ac4adf336f5c2c2a2de505befee`.
  It is a review artifact, not a published package or managed installation.
- Headless Chrome checked the real embedded app using synthetic local accounts
  and isolated NTFS data, with helper downloads/transcoding disabled. Root `/`
  and Workspace `/my-tube/` both load; header SVG and favicon bytes match the
  repository logo. No unexpected HTTP failures or JavaScript errors occurred.
  Missing optional channel artwork correctly uses fallback artwork.
- Reviewed screenshots at desktop 1440px, phone 390px and narrow phone 320px,
  covering light/dark home, dark channel fallback and light upload screen.
  No horizontal overflow; theme switching, home navigation and keyboard-visible
  brand-button focus passed. No playback/import/backend regression suite or
  installed-owner fixture was used for this visual change.
- Retained seven `docs/screenshots/workspace-branding-*.png` previews. The
  disposable native/browser processes were closed and test data removed.

## Resume

Review the local diff and previews. If publication is subsequently requested,
follow the current app lifecycle, allocate a new immutable app version and use
the supported managed update path. Do not claim this styling is already present
on installed machines; no owner installation/data/credentials were changed.

## Publication authorized

The user subsequently requested push, release and installation. Prepare a new
immutable preview6 app release, refresh the public channel/component pin, then
use the normal My Tube-only updater on both existing PCs. Both currently report
Workspace113 and Tube5. Preserve unrelated versions/selections, including the
separately pending Finance3 update on PC2. No suite release/Android rebuild is
needed for this independent app update. Publication and installation are pending.

## Release checks

Frontend rebuild, full Go suite and vet passed; stripped Windows x64 and Linux
x64 cross-builds completed with Go1.27.3. The pre-existing migration test tried
to UPDATE every migration key to99; after schema3 this violates its primary key,
leaving the fixture unchanged. It now checks a successful INSERT of a future
migration and closes any unexpectedly accepted store. Production migration logic
is unchanged. Final preview6 binary visual checks and packaging follow.

The suite managedSeed path also ignored the existing myTube pin, so a scoped
My Tube replacement adapter and ownership/tamper regression coverage are being
added in an isolated suite worktree. This is required so ordinary future builds
consume the published app6 instead of retaining the seed's older Tube payload.

Final stripped preview6 embedded-binary checks passed at root and /my-tube/,
including exact logo/favicon bytes, desktop/phone light/dark rendering, 320px
overflow checks, navigation and keyboard focus. Previews now show version6.
