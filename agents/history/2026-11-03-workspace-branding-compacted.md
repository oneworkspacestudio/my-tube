# My Tube One Workspace branding (2026-11-03)

## Scope and state

The user requested One Workspace design-system colours/branding and My Tube's
existing repository logo. Started from clean `main` at `86694e2`, equal to
`origin/main`. The initial source change was followed by explicit authorization to push, release
and install. Current state: preview6 is public and both PCs are installed and
verified. Existing preview5 assets remain unchanged; delivery details follow.

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

## Initial review handoff (superseded by delivery below)

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

## Delivery complete

Published immutable [v0.1.0-preview.6](https://github.com/oneworkspacestudio/my-tube/releases/tag/v0.1.0-preview.6)
from e7d562b948a390c7d00641d21b2187457bb59a0d. All13 public assets match local
size/SHA256; anonymous Windows archive download matches
2b75f650cf96b2c4e4b223f77a122333027e04d7951d873c3085d0ad31c830dd.
Final stripped EXE SHA256561cc2dd1248b9279bce11f246a9a078a30b59de2720de6bb7faa881e613dc24.
Suite public channel/component pin52785eb includes6; anonymous exact raw-main
channel check passes with every preceding release/asset/withholding retained.

The suite ordinary rebuild now consumes Tube6:37 apps/451 archived payloads,
all size/hash checked, catalog audit passed. Candidate planner/catalog/distribution
and two component boundary tests pass. Local suite114 is QA only; no Workspace
release or Android publication occurred. The independent app updater is the
installed delivery path. GitHub Actions stayed disabled in both repositories.

PC2 normal Tube-only update5->6 completed in2m41s; primary2->6 completed in5m52s.
Both retain Workspace113 and their latest verified recovery copies. Fresh update
checks show Tube6 up to date and every unrelated displayed installed version
unchanged from its pre-apply baseline. The primary's four owned payloads match
all release hashes; both deployments serve exact tested SVG/CSS bytes and show
the repo logo, blue palette and footer6. Existing PC2 Imports tab refreshed.
No extra PC2 restart was performed after completion.

The initial primary attempt failed before preparation while a separate Finance4
CLI update held update.lock during recovery cleanup. Read-only Windows Restart
Manager inspection identified that owner. Waited for its normal exit, refreshed
and reselected only Tube, then completed one successful normal apply; no lock or
process was forced. Primary Finance4/MyAI20 and PC2 Finance2/MyAI23 were retained.
Pre-existing unrelated update/review choices were not applied.

Private receipts, served-asset checks and screenshots are in the task tooling
artifact directory outside Git. Synthetic NTFS data and the initial failed-test
residue were removed; no task preview services remain. No required release or
installation work remains. Physical-phone, Linux-runtime and reboot acceptance
remain unclaimed; this is an unsigned preview, not a stable-release claim.
