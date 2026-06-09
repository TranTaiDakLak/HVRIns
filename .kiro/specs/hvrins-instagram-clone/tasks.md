# Implementation Plan: HVRIns Instagram Clone

## Overview

This plan migrates the source Wails project at `e:\WEMAKE\NullCoreSummer` (Go module `HVR`, app
`HaVu`) into the Instagram-branded `HVRIns` project rooted at the current workspace
`e:\WEMAKE\HVRIns`, then rebrands it visually and builds it. Work proceeds strictly in build order:
copy the tree first, transform the Go code, get the backend compiling, regenerate Wails bindings,
rebrand the frontend, build the frontend, run the full Wails build, then verify branding.

All transformation rules use the corrected source identifiers from the design (module `HVR`,
package `facebook`, env var `HVR_DATA_DIR`), not the Build Guide's `HVRFb` placeholder. Languages:
Go (backend + Go transform helpers and property tests), JavaScript/TypeScript with `fast-check`
(Vue color transforms and property tests), Python/Pillow (icon generation).

## Tasks

- [x] 1. Migrate the source tree into the HVRIns target
  - [x] 1.1 Copy the source tree into the target, preserving target assets
    - Recursively copy `e:\WEMAKE\NullCoreSummer` into `e:\WEMAKE\HVRIns`
    - Exclude `.git/`, `*.exe`, `build/bin/`, `logs/`, `result/`, `bin/dev/`, `frontend/node_modules/`, `frontend/dist/`, `tmp/`, `*.log`, and any source `.kiro/`
    - Do NOT overwrite or delete the pre-existing `NVRINS_BUILD_GUIDE.md` or `.kiro/` in the target
    - _Requirements: 2.1_

  - [ ]* 1.2 Write an integration test asserting copy fidelity and exclusions
    - Assert non-excluded files were copied, excluded artifacts are absent, and `NVRINS_BUILD_GUIDE.md` / `.kiro/` survived
    - _Requirements: 2.1_

- [x] 2. Rename project identity to HVRIns
  - [x] 2.1 Edit `go.mod` and `wails.json` identity fields
    - Set `go.mod` module declaration to `module HVRIns`
    - Set `wails.json` `name` to `HVRIns` and `outputfilename` to `HVRIns`
    - Apply the guide's `NVRIns` → `HVRIns` example-name substitution wherever it appears
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.7_

  - [x] 2.2 Implement the identity-rename precondition guard
    - Before writing, stat/open `go.mod` and `wails.json`; if missing or unwritable, abort the rename, leave all target files unchanged, and return an error naming the affected file
    - _Requirements: 1.5_

  - [ ]* 2.3 Write unit tests for identity rename and guard
    - Assert exact resulting values and zero remaining source-module identifiers in `go.mod`/`wails.json`; drive the missing/unwritable path and assert no-mutation + correct error
    - _Requirements: 1.5, 1.6_

- [ ] 3. Rename the facebook package and apply the Go transformation engine
  - [x] 3.1 Implement the Go text-transformation functions (rules G1–G4)
    - Pure `(fileText) -> fileText` functions: G1 `"HVR/internal/facebook` → `"HVRIns/internal/instagram`; G2 remaining `"HVR/` → `"HVRIns/`; G3 `package facebook` → `package instagram`; G4 `facebook.<Upper>` → `instagram.<Upper>`; run G1 before G2; never touch `facebook.com`
    - _Requirements: 2.2, 2.3, 2.4, 2.5, 2.6_

  - [ ]* 3.2 Write property test for package/symbol transform
    - **Property 1: Go transform rewrites package/symbol while preserving `facebook.com`**
    - Tag: `Feature: hvrins-instagram-clone, Property 1`
    - Use a Go PBT library (`testing/quick` or `pgregory.net/rapid`), minimum 100 generated inputs
    - **Validates: Requirements 2.2, 2.5, 2.6**

  - [ ]* 3.3 Write property test for import-prefix precedence
    - **Property 2: Import-prefix rewrite respects facebook-subpath precedence**
    - Tag: `Feature: hvrins-instagram-clone, Property 2`
    - Minimum 100 generated inputs
    - **Validates: Requirements 2.3, 2.4**

  - [x] 3.4 Move `internal/facebook` to `internal/instagram` with guards
    - Move the directory byte-for-byte; if source `internal/facebook` is absent, abort and report the missing source; if `internal/instagram` already exists, abort and report the destination conflict
    - Run the move before applying text transforms
    - _Requirements: 2.1, 2.7, 2.8_

  - [ ] 3.5 Apply the Go transforms across all `*.go` files
    - Run G1–G4 over every Go source file in the migrated tree (idempotent); confirm zero remaining `"HVR/` import prefixes and zero `package facebook` declarations
    - _Requirements: 2.2, 2.3, 2.4, 2.5, 2.6_

  - [ ]* 3.6 Write unit tests for the directory-move guards
    - Drive the missing-source and destination-conflict paths; assert no-mutation + correct error
    - _Requirements: 2.7, 2.8_

- [ ] 4. Stub Facebook-specific platforms
  - [ ] 4.1 Add the `StatusUnsupportedPlatform` constant
    - Add a single `unsupported platform` status value to the existing status enumeration in `internal/instagram/status.go`
    - _Requirements: 3.2_

  - [ ] 4.2 Replace s545–s560v3 handler bodies with stubs
    - Keep `register/web`, `verify/web`, `register/android`, `verify/android` as named, compilable handler structures; replace Facebook protocol bodies with stubs that return `StatusUnsupportedPlatform`, no error, and mutate neither arguments nor state
    - _Requirements: 3.1, 3.2, 3.3_

  - [ ]* 4.3 Write property test for stub dispatch
    - **Property 3: Stubbed platforms return unsupported status without side effects**
    - Tag: `Feature: hvrins-instagram-clone, Property 3`
    - Generate platform identifiers across `s545..s560v3` and arbitrary inputs; snapshot pre-state for the no-side-effect assertion; minimum 100 generated inputs
    - **Validates: Requirements 3.2, 3.3**

  - [ ] 4.4 Annotate Facebook session tokens with TODO comments
    - Add `// TODO(instagram): ...` to `fb_dtsg`, `jazoest`, `lsd`, `datr`, `c_user`, keeping each existing value unchanged
    - _Requirements: 3.4_

  - [ ] 4.5 Annotate Facebook user-agent pool entries with TODO comments
    - Add `// TODO(instagram): ...` to `FBAN/FB4A` and `FBPN/com.facebook.katana`, keeping each existing value unchanged
    - _Requirements: 3.5_

- [ ] 5. Migrate runtime path env var
  - [ ] 5.1 Rename `HVR_DATA_DIR` to `HVRINS_DATA_DIR`
    - Rename the env var identifier in `datadir.go` and all references; leave data-dir resolution logic and the relative `logs/`, `result/`, `Config/` join segments intact; confirm zero remaining source env var references
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [ ]* 5.2 Write integration test for runtime path resolution
    - Set `HVRINS_DATA_DIR` and assert `logs/`, `result/`, `Config/` resolve under it with no reference to the source env var name
    - _Requirements: 4.5_

- [ ] 6. Checkpoint - backend compiles
  - Run `go build ./...` at the project root and resolve any errors so every package compiles
  - Ensure all tests pass, ask the user if questions arise.
  - _Requirements: 14.1, 14.2_

- [ ] 7. Regenerate Wails bindings
  - [ ] 7.1 Regenerate the Wails bindings
    - Run `wails generate module` (or first `wails dev`) after the Go type/package changes so `frontend/wailsjs/` references `instagram` symbols
    - _Requirements: 5.1, 5.2_

  - [ ] 7.2 Implement binding-regeneration failure handling
    - On non-zero CLI exit, report the binding-generation error and retain the previously generated bindings unchanged
    - _Requirements: 5.4_

  - [ ]* 7.3 Write a test asserting binding symbol migration
    - Grep generated bindings: assert `instagram` symbols are referenced and zero `facebook` symbol references remain
    - _Requirements: 5.2, 5.3_

- [x] 8. Generate the Instagram application icon
  - [x] 8.1 Write the Pillow icon script and generate `build/appicon.png`
    - 1024×1024 PNG, rounded corners radius 160; 135° gradient through `#405DE6 → #833AB4 → #E1306C → #FD1D1D → #FCAF45`; centered white `HVRIns`; subtitle `Ha Vu VIP PRO` below it at 60% opacity
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [x] 8.2 Derive `build/windows/icon.ico`
    - Produce an ICO containing exactly sizes 16/32/48/64/128/256 from `build/appicon.png`
    - _Requirements: 6.5_

  - [x] 8.3 Implement icon-generation failure handling
    - On failure, report the failing asset and leave any existing icon file unchanged
    - _Requirements: 6.6_

  - [ ]* 8.4 Write asset tests for the generated icons
    - Assert PNG dimensions/corner radius and sampled gradient stops; assert the ICO contains exactly the six sizes
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 9. Apply the Instagram design token palette
  - [x] 9.1 Edit `frontend/src/styles/tokens.css`
    - Set `--brand-primary:#E1306C`, `--accent:#E1306C`, `--brand-gradient:linear-gradient(135deg,#833AB4,#E1306C,#FD1D1D)`, `--sidebar-bg:#0e0c18` (plus hover/bg/border aliases)
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [x] 9.2 Edit `frontend/src/styles/light.css`
    - Set light-theme `--brand-primary:#E1306C` and `--accent:#E1306C` (plus sidebar/grid/input-focus accents)
    - _Requirements: 7.5, 7.6_

  - [ ]* 9.3 Write tests asserting token values
    - Assert exact resulting token values in `tokens.css` and `light.css`
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6_

- [x] 10. Restyle the sidebar
  - [x] 10.1 Edit `frontend/src/components/shell/AppSidebar.vue`
    - Background `var(--sidebar-bg)`; right border 3px solid transparent with a vertical `border-image` gradient through `#405DE6, #833AB4, #E1306C, #FD1D1D, #FCAF45`
    - _Requirements: 8.1, 8.2, 8.3_

  - [ ]* 10.2 Write a snapshot test for the sidebar
    - Snapshot the rendered background and vertical gradient right border
    - _Requirements: 8.1, 8.2, 8.3_

- [x] 11. Replace the title bar logo
  - [x] 11.1 Edit `frontend/src/components/shell/AppTitleBar.vue`
    - Replace the logo SVG with an Instagram camera (rounded-square outline + centered circle + upper dot); gradient stroke `linear-gradient(135deg,#833AB4,#E1306C,#FD1D1D)`; render it in the title bar header region
    - _Requirements: 9.1, 9.2, 9.3_

  - [ ]* 11.2 Write a snapshot test for the title bar logo
    - Snapshot the camera SVG structure and gradient stroke
    - _Requirements: 9.1, 9.2, 9.3_

- [ ] 12. Build the single-line footer with Teleport slot
  - [ ] 12.1 Edit `frontend/src/components/shell/AppStatusBar.vue`
    - Remove the `stats` prop; add `#status-bar-page-slot`; show CPU% and RAM% (0–100); add exactly two icon-only buttons each with a non-empty `data-tip` driving a pure-CSS `::after` tooltip
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

  - [x] 12.2 Wrap the Accounts stats bar in a Teleport
    - In `frontend/src/pages/AccountsPage.vue`, wrap the stats bar in `<Teleport to="#status-bar-page-slot">`
    - _Requirements: 11.6_

  - [ ]* 12.3 Write a component test for Teleport behavior
    - Mount with and without page slot content; assert the footer shows page stats alongside CPU/RAM + two buttons when present, and only CPU/RAM + two buttons when absent
    - _Requirements: 11.7, 11.8_

- [x] 13. Apply gradient styling to action buttons
  - [x] 13.1 Edit `frontend/src/modules/accounts/components/AccountsToolbar.vue`
    - Run btn `linear-gradient(135deg,#833AB4,#E1306C,#FD1D1D)`; Stop btn `linear-gradient(135deg,#E1306C,#FD1D1D)`; Stopping btn `linear-gradient(135deg,#405DE6,#833AB4)`
    - _Requirements: 10.1, 10.2, 10.3_

  - [x] 13.2 Edit `frontend/src/components/ui/BaseButton.vue`
    - Primary variant background `var(--brand-gradient)` with a solid `#E1306C` fallback when the token fails to resolve
    - _Requirements: 10.4, 10.7_

  - [x] 13.3 Edit the Accounts page CTA
    - In `frontend/src/pages/AccountsPage.vue`, set `.accounts-page__cta` background to `var(--brand-gradient)`
    - _Requirements: 10.5_

  - [ ]* 13.4 Write snapshot + contrast tests for gradient buttons
    - Snapshot button gradients/fallback; assert foreground vs lightest gradient stop ≥ 4.5:1
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7_

- [ ] 14. Replace hard-coded blue/cyan colors
  - [x] 14.1 Implement the Vue color-transformation functions (rules C1–C3)
    - Pure functions over `.vue` text: C1 `#4fc3f7` → `var(--accent)` (skip if inside `var(--accent, ...)`); C2 `#3b82f6` → `var(--accent)` (skip if inside `var(--accent-hover, ...)`); C3 `rgba(79,195,247,A)` → `rgba(225,48,108,A)` preserving alpha and tolerating internal whitespace; all case-insensitive
    - _Requirements: 12.1, 12.2, 12.3, 12.4_

  - [ ]* 14.2 Write property test for guarded hex replacement
    - **Property 4: Hex color replacement is guarded and complete**
    - Tag: `Feature: hvrins-instagram-clone, Property 4`
    - Use `fast-check`; generate guarded and unguarded occurrences in varied case; minimum 100 generated inputs
    - **Validates: Requirements 12.1, 12.2, 12.4**

  - [ ]* 14.3 Write property test for rgba alpha preservation
    - **Property 5: rgba color replacement preserves alpha**
    - Tag: `Feature: hvrins-instagram-clone, Property 5`
    - Use `fast-check`; generate `rgba(79, 195, 247, A)` with random alpha and internal whitespace; minimum 100 generated inputs
    - **Validates: Requirements 12.3, 12.4**

  - [ ] 14.4 Apply the color transforms to the three target files
    - Apply C1–C3 to `InteractionSetupPage.vue`, `GeneralSettingsPage.vue`, `AuthSourcePanel.vue`; on any unreadable/unwritable file, abort that file's replacements, leave it unchanged, and report the affected file; verify no unguarded `#4fc3f7`/`#3b82f6`/`rgba(79,195,247,...)` remains
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5_

- [ ] 15. Build the frontend
  - [ ] 15.1 Run the frontend build
    - Run `npm install` then `npm run build` in `frontend/`; resolve compile errors; success = exit 0 with emitted dist artifacts
    - _Requirements: 13.1, 13.2, 13.3_

- [ ] 16. Produce the executable
  - [ ] 16.1 Run the full Wails build
    - Run `wails build` at the project root; success = exit 0 producing `build/bin/HVRIns.exe`; on non-zero exit, report the build failure and do not report success for an absent/incomplete binary
    - _Requirements: 15.1, 15.2, 15.3_

- [ ] 17. Verify the rebrand
  - [ ]* 17.1 Write aggregate branding snapshot/smoke tests
    - Snapshot-verify the Instagram icon/gradient, title-bar camera logo, `#0e0c18` sidebar with vertical gradient border, single-line footer, `--brand-gradient` action buttons, `--accent` highlights, and that no leftover blue/cyan literals remain; assert the Accounts stats render in `#status-bar-page-slot` when on the Accounts page and the slot is empty (CPU/RAM + two buttons only) when navigated away
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_

- [ ] 18. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional test sub-tasks and can be skipped for a faster skeleton.
- The migration transforms (Go rules G1–G4, Vue rules C1–C3) are pure, idempotent functions; re-running the pipeline after a fixed failure converges to the fully migrated state.
- All transform rules target the corrected source identifiers (module `HVR`, package `facebook`, env var `HVR_DATA_DIR`) per the design's Discrepancies section, not the Build Guide's `HVRFb` placeholder.
- Property tests cover only the input-varying logic (Properties 1–5); deterministic edits, asset generation, UI markup, and build outcomes are covered by example, snapshot, integration, and smoke tests.
- Each property test must run a minimum of 100 generated inputs and carry its `Feature: hvrins-instagram-clone, Property N` tag.
- Requirement 16's manual visual confirmation via `wails dev` is outside automated coding scope; task 17.1 covers its intent through automated snapshot/smoke tests.

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["2.1", "3.1", "14.1", "8.1", "9.1", "9.2", "11.1", "10.1", "12.2"] },
    { "id": 2, "tasks": ["2.2", "3.4", "8.2", "8.3", "13.1", "13.2", "13.3"] },
    { "id": 3, "tasks": ["1.2", "2.3", "3.2", "3.3", "3.5", "14.2", "14.3", "12.1"] },
    { "id": 4, "tasks": ["3.6", "4.1", "8.4", "9.3", "10.2", "11.2", "12.3", "14.4"] },
    { "id": 5, "tasks": ["4.2", "4.4", "4.5"] },
    { "id": 6, "tasks": ["4.3", "5.1"] },
    { "id": 7, "tasks": ["5.2", "7.1"] },
    { "id": 8, "tasks": ["7.2", "7.3"] },
    { "id": 9, "tasks": ["13.4", "15.1"] },
    { "id": 10, "tasks": ["16.1"] },
    { "id": 11, "tasks": ["17.1"] }
  ]
}
```
