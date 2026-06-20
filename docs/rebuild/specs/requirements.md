# Requirements Document

## Introduction

This feature covers cloning the existing HVRFb application (a Wails desktop app with a Go backend and a Vue 3 frontend, originally built for Facebook automation) into a new project named **HVRIns**, rebranded for Instagram. The work follows the process described in `NVRINS_BUILD_GUIDE.md`, adapting every example name from "NVRIns" to "HVRIns".

The effort is divided into three phases:

1. **Code migration** — copy the source repository, rename the Go module and Wails app to `HVRIns`, rename the `internal/facebook` package to `internal/instagram`, bulk-replace imports and symbols, stub Facebook-specific platforms, mark Facebook-specific tokens and user-agent pools as TODO placeholders for Instagram equivalents, update runtime paths, and regenerate Wails bindings.
2. **Visual rebrand** — replace the app icon with an Instagram gradient icon, update design tokens and palette to Instagram colors, restyle sidebar, title bar, action buttons, footer, and replace hard-coded blue/cyan colors with the accent token.
3. **Build and verify** — produce a compiling, build-able skeleton (`HVRIns.exe`) and verify the rebrand visually.

The scope of Phase 1 is a build-able skeleton only. Real Instagram protocol flows (endpoints, headers, cookies, token parsers) are intentionally NOT ported in this feature; Facebook-specific platforms return an "unsupported platform" result and Facebook-specific tokens and user-agent pools remain as documented TODO placeholders.

## Glossary

- **HVRFb**: The existing source project (Facebook automation Wails desktop app) located outside the current workspace. The migration baseline.
- **HVRIns**: The new target project (Instagram-branded clone) created by this feature, rooted at the current workspace.
- **Build_Guide**: The document `NVRINS_BUILD_GUIDE.md`, the authoritative process reference for this feature.
- **Migration_Process**: The set of code transformation steps that convert HVRFb source into HVRIns source (Phase 1).
- **Rebrand_Process**: The set of visual changes that apply Instagram branding to HVRIns (Phase 2).
- **Build_System**: The combination of npm (frontend), the Go compiler, and the Wails CLI that compiles HVRIns into an executable (Phase 3).
- **Wails_Bindings**: The generated Go-to-frontend binding code produced by the Wails CLI.
- **Facebook_Platform**: Any platform variant specific to the Facebook API version, identified as `s545` through `s560v3`.
- **Stubbed_Platform**: A Facebook_Platform whose implementation returns an "unsupported platform" result instead of Facebook logic.
- **Instagram_Palette**: The Instagram brand color set: `#405DE6`, `#833AB4`, `#E1306C`, `#FD1D1D`, `#FCAF45`, with primary `#E1306C` and gradient `linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D)`.
- **Accent_Token**: The CSS custom property `--accent`, set to `#E1306C`.
- **Status_Bar**: The single-line application footer component (`AppStatusBar.vue`).
- **Page_Slot**: The Teleport target element `#status-bar-page-slot` rendered inside the Status_Bar.
- **Runtime_Path**: A filesystem path used by the running application for logs, results, or configuration.

## Requirements

### Requirement 1: Project identity rename

**User Story:** As a developer, I want the cloned project to identify itself as HVRIns, so that the module, app name, and output binary are distinct from HVRFb.

#### Acceptance Criteria

1. WHEN the project identity rename is executed, THE Migration_Process SHALL set the `go.mod` module declaration to the exact value `module HVRIns`.
2. WHEN the project identity rename is executed, THE Migration_Process SHALL set the `wails.json` `name` field to the exact value `HVRIns`.
3. WHEN the project identity rename is executed, THE Migration_Process SHALL set the `wails.json` `outputfilename` field to the exact value `HVRIns`.
4. WHERE the Build_Guide uses the example name `NVRIns`, THE Migration_Process SHALL replace every occurrence of `NVRIns` with the name `HVRIns`.
5. IF a required configuration file (`go.mod` or `wails.json`) is missing or cannot be written during the rename, THEN THE Migration_Process SHALL abort the rename, leave all target files unchanged, and return an error indicating the affected file.
6. WHEN the project identity rename completes, THE Migration_Process SHALL ensure that zero occurrences of the prior module name `HVRFb` remain in `go.mod` and `wails.json`.
7. WHEN a Wails build is run after the rename completes, THE Migration_Process SHALL produce an output binary named exactly `HVRIns.exe`.

### Requirement 2: Package rename from facebook to instagram

**User Story:** As a developer, I want the internal Facebook package renamed to Instagram, so that the package structure reflects the new target platform.

#### Acceptance Criteria

1. THE Migration_Process SHALL rename the directory `internal/facebook` to `internal/instagram`, preserving all contained files and subdirectories without content modification.
2. THE Migration_Process SHALL replace every Go package declaration `package facebook` with `package instagram`, such that zero `package facebook` declarations remain after completion.
3. WHEN a Go source file imports a path beginning with `HVRFb/internal/facebook`, THE Migration_Process SHALL replace the prefix `HVRFb/internal/facebook` with `HVRIns/internal/instagram`.
4. WHEN a Go source file imports a path beginning with `HVRFb/` that does NOT begin with `HVRFb/internal/facebook`, THE Migration_Process SHALL replace the prefix `HVRFb/` with `HVRIns/`.
5. WHEN a Go source file references a qualifier matching `facebook.` immediately followed by an uppercase ASCII letter (A through Z), THE Migration_Process SHALL replace the `facebook.` qualifier with `instagram.`.
6. THE Migration_Process SHALL preserve, without modification, every occurrence of the literal substring `facebook.com` within string literal values.
7. IF the directory `internal/facebook` does not exist when the rename operation begins, THEN THE Migration_Process SHALL abort the operation, leave all files unchanged, and produce an error indication identifying the missing source directory.
8. IF the directory `internal/instagram` already exists when the rename operation begins, THEN THE Migration_Process SHALL abort the operation, leave all files unchanged, and produce an error indication identifying the destination conflict.

### Requirement 3: Stub Facebook-specific platforms

**User Story:** As a developer, I want Facebook-specific platforms stubbed, so that the project builds without porting Facebook protocol logic that has no Instagram meaning.

#### Acceptance Criteria

1. THE Migration_Process SHALL retain `register/web`, `verify/web`, `register/android`, and `verify/android` as named, compilable handler structures whose Facebook protocol implementation bodies are removed and replaced with stub bodies.
2. WHEN a Facebook_Platform in the range `s545` through `s560v3` is invoked, THE Stubbed_Platform SHALL return a result whose status indicates `unsupported platform` without executing any Facebook protocol logic.
3. IF a Facebook_Platform in the range `s545` through `s560v3` is invoked, THEN THE Stubbed_Platform SHALL leave any input arguments and application state unchanged and SHALL NOT raise a compilation or runtime error.
4. THE Migration_Process SHALL annotate each of the Facebook-specific tokens `fb_dtsg`, `jazoest`, `lsd`, `datr`, and `c_user` with an inline TODO comment indicating that an Instagram equivalent is pending, while retaining the existing token reference unchanged.
5. THE Migration_Process SHALL annotate each of the Facebook-specific user-agent pool entries `FBAN/FB4A` and `FBPN/com.facebook.katana` with an inline TODO comment indicating that an Instagram equivalent is pending, while retaining the existing entry value unchanged.

### Requirement 4: Update runtime paths

**User Story:** As a developer, I want runtime paths to use the HVRIns name, so that the cloned app reads and writes its own log, result, and config directories.

#### Acceptance Criteria

1. WHEN the logs Runtime_Path contains the path segment `HVRFb` under the application data directory, THE Migration_Process SHALL replace only that `HVRFb` segment with `HVRIns`, leaving the application data directory prefix and all trailing path components unchanged.
2. WHEN the result Runtime_Path contains the path segment `HVRFb` under the documents directory, THE Migration_Process SHALL replace only that `HVRFb` segment with `HVRIns`, leaving the documents directory prefix and all trailing path components unchanged.
3. WHEN the config Runtime_Path contains the path segment `HVRFb` under the documents directory, THE Migration_Process SHALL replace only that `HVRFb` segment with `HVRIns`, leaving the documents directory prefix and all trailing path components unchanged.
4. THE Migration_Process SHALL replace every occurrence of the `HVRFb` Runtime_Path segment for logs, result, and config such that zero `HVRFb` Runtime_Path segments remain in the migrated source.
5. AFTER the Migration_Process completes, THE HVRIns SHALL read and write its logs, result, and config under the `HVRIns` Runtime_Path segments and SHALL NOT reference any `HVRFb` Runtime_Path segment.

### Requirement 5: Regenerate Wails bindings

**User Story:** As a developer, I want Wails bindings regenerated after Go type changes, so that the frontend uses the current Go symbols instead of stale HVRFb bindings.

#### Acceptance Criteria

1. WHEN Go types or packages have changed during the Migration_Process, THE Migration_Process SHALL regenerate the Wails_Bindings.
2. WHEN the Wails_Bindings are regenerated, THE Wails_Bindings SHALL reference the `instagram` package symbols invoked by the frontend.
3. WHEN the Wails_Bindings are regenerated, THE Wails_Bindings SHALL contain zero references to `facebook` package symbols.
4. IF regeneration of the Wails_Bindings fails, THEN THE Migration_Process SHALL report an error indicating the binding generation failure and retain the previously generated Wails_Bindings unchanged.

### Requirement 6: Application icon rebrand

**User Story:** As a user, I want the app icon to show the Instagram gradient, so that the application is visually identified as the Instagram edition.

#### Acceptance Criteria

1. THE Rebrand_Process SHALL produce `build/appicon.png` as a 1024-by-1024 pixel PNG with rounded corners of radius 160 pixels on all four corners.
2. THE Rebrand_Process SHALL render the `build/appicon.png` background as a 135-degree diagonal linear gradient running from top-left to bottom-right through the Instagram_Palette stops in the order `#405DE6`, `#833AB4`, `#E1306C`, `#FD1D1D`, then `#FCAF45`.
3. THE Rebrand_Process SHALL render the application name text `HVRIns` centered both horizontally and vertically on `build/appicon.png` in white at 100% opacity.
4. THE Rebrand_Process SHALL render the subtitle `Ha Vu VIP PRO` immediately below the application name on `build/appicon.png` in white at 60% opacity.
5. THE Rebrand_Process SHALL produce `build/windows/icon.ico` containing exactly the six square sizes 16, 32, 48, 64, 128, and 256 pixels derived from `build/appicon.png`.
6. IF generation of `build/appicon.png` or `build/windows/icon.ico` fails, THEN THE Rebrand_Process SHALL report an error identifying the asset that failed and leave any existing icon file unchanged.

### Requirement 7: Design token palette rebrand

**User Story:** As a user, I want the design tokens set to Instagram colors, so that interactive elements consistently render in Instagram pink.

#### Acceptance Criteria

1. THE Rebrand_Process SHALL set `--brand-primary` to `#E1306C` in `tokens.css`.
2. THE Rebrand_Process SHALL set the Accent_Token `--accent` to `#E1306C` in `tokens.css`.
3. THE Rebrand_Process SHALL set `--brand-gradient` to `linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D)` in `tokens.css`.
4. THE Rebrand_Process SHALL set `--sidebar-bg` to `#0e0c18` in `tokens.css`.
5. THE Rebrand_Process SHALL set the light-theme `--brand-primary` token in `light.css` to `#E1306C`.
6. THE Rebrand_Process SHALL set the light-theme Accent_Token `--accent` in `light.css` to `#E1306C`.

### Requirement 8: Sidebar visual separation

**User Story:** As a user, I want the sidebar visually distinct from the header, so that the navigation area reads as a separate Instagram-accented region.

#### Acceptance Criteria

1. THE Rebrand_Process SHALL set the `AppSidebar.vue` background to the `--sidebar-bg` token value (`#0e0c18`).
2. THE Rebrand_Process SHALL apply a right border on `AppSidebar.vue` with a width of 3 pixels and a solid border style so that the `border-image` gradient is rendered.
3. THE Rebrand_Process SHALL set the `AppSidebar.vue` right `border-image` to a top-to-bottom (vertical) linear gradient through the Instagram_Palette stops `#405DE6`, `#833AB4`, `#E1306C`, `#FD1D1D`, and `#FCAF45`.

### Requirement 9: Title bar Instagram logo

**User Story:** As a user, I want an Instagram-style logo in the title bar, so that the window header reflects the Instagram brand.

#### Acceptance Criteria

1. THE Rebrand_Process SHALL replace the existing `AppTitleBar.vue` logo SVG with an Instagram camera SVG that contains exactly one rounded-square outer outline, one centered circle, and one dot positioned in the upper region of the rounded square.
2. THE Rebrand_Process SHALL apply the Instagram_Palette gradient `linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D)` as the stroke color of the `AppTitleBar.vue` logo SVG shapes defined in criterion 1.
3. WHEN the `AppTitleBar.vue` component is rendered, THE AppTitleBar SHALL display the Instagram camera logo SVG in the title bar header region.

### Requirement 10: Action button gradients

**User Story:** As a user, I want primary and run/stop action buttons styled with the Instagram gradient, so that calls to action match the new brand.

#### Acceptance Criteria

1. THE Rebrand_Process SHALL set the CSS background of the `AccountsToolbar.vue` Run button to the gradient `linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D)`, with all three color stops present and in the stated order.
2. THE Rebrand_Process SHALL set the CSS background of the `AccountsToolbar.vue` Stop button to the gradient `linear-gradient(135deg, #E1306C, #FD1D1D)`, with both color stops present and in the stated order.
3. THE Rebrand_Process SHALL set the CSS background of the `AccountsToolbar.vue` Stopping button to the gradient `linear-gradient(135deg, #405DE6, #833AB4)`, with both color stops present and in the stated order.
4. THE Rebrand_Process SHALL set the CSS background of the `BaseButton.vue` primary variant to the resolved value of the `--brand-gradient` token.
5. THE Rebrand_Process SHALL set the CSS background of the `AccountsPage.vue` call-to-action button to the resolved value of the `--brand-gradient` token.
6. THE Rebrand_Process SHALL render the foreground text and icons of each gradient-styled button (criteria 1-5) with a color that maintains a minimum contrast ratio of 4.5:1 against the lightest color stop of the applied gradient.
7. IF the `--brand-gradient` token is undefined or fails to resolve at render time, THEN THE Rebrand_Process SHALL apply a solid `#E1306C` background to the affected button so the call to action remains visible.

### Requirement 11: Single-line footer with Teleport slot

**User Story:** As a user, I want a single-line footer that shows page-specific stats and utility buttons, so that footer information is consolidated and contextual.

#### Acceptance Criteria

1. THE Rebrand_Process SHALL remove the `stats` prop from `AppStatusBar.vue`.
2. THE Rebrand_Process SHALL add the Page_Slot element `#status-bar-page-slot` to `AppStatusBar.vue`.
3. WHILE the application is running, THE Status_Bar SHALL display the current CPU usage and RAM usage each as a percentage value in the range 0 to 100.
4. THE Status_Bar SHALL display exactly two icon-only buttons, each carrying a non-empty `data-tip` attribute that defines its CSS tooltip text.
5. WHEN a user hovers the pointer over one of the two icon-only buttons, THE Status_Bar SHALL display a CSS tooltip showing that button's `data-tip` attribute text.
6. THE Rebrand_Process SHALL wrap the `AccountsPage.vue` stats bar in a `Teleport` element targeting `#status-bar-page-slot`.
7. WHEN the active page provides Page_Slot content, THE Status_Bar SHALL render that content within the `#status-bar-page-slot` element alongside the resource usage values and the two utility buttons.
8. WHEN the active page provides no Page_Slot content, THE Status_Bar SHALL display only the CPU usage value, the RAM usage value, and the two utility buttons.

### Requirement 12: Replace hard-coded blue and cyan colors

**User Story:** As a user, I want hard-coded blue and cyan colors replaced with the accent token, so that all highlighted elements render in Instagram pink.

#### Acceptance Criteria

1. WHEN the hex color literal `#4fc3f7` (matched case-insensitively, covering both 6-digit forms such as `#4FC3F7`) appears in any of the three files `InteractionSetupPage.vue`, `GeneralSettingsPage.vue`, or `AuthSourcePanel.vue`, and the literal is not already inside an existing `var(--accent, ...)` fallback expression, THE Rebrand_Process SHALL replace each such occurrence with `var(--accent)`.
2. WHEN the hex color literal `#3b82f6` (matched case-insensitively) appears in any of the three files `InteractionSetupPage.vue`, `GeneralSettingsPage.vue`, or `AuthSourcePanel.vue`, and the literal is not already inside an existing `var(--accent-hover, ...)` fallback expression, THE Rebrand_Process SHALL replace each such occurrence with `var(--accent)`.
3. WHEN an `rgba(79, 195, 247, A)` color appears in any of the three listed files, where the three leading components equal `79`, `195`, and `247` regardless of internal whitespace (for example `rgba(79,195,247,A)` or `rgba(79, 195, 247, A)`) and `A` is the existing alpha value, THE Rebrand_Process SHALL replace the three numeric red-green-blue components with `225`, `48`, and `108` while preserving the original alpha value `A` unchanged.
4. WHEN the Rebrand_Process completes its replacements across the three listed files, THE Rebrand_Process SHALL verify that no remaining occurrence of `#4fc3f7`, `#3b82f6` (case-insensitive), or `rgba(79, 195, 247, ...)` (any internal whitespace) exists outside an existing `var(--accent, ...)` or `var(--accent-hover, ...)` fallback expression in those files.
5. IF any of the three listed files cannot be read or written during the Rebrand_Process, THEN THE Rebrand_Process SHALL abort all replacements for that file, leave the file content unchanged, and produce an error indication identifying the affected file.

### Requirement 13: Build the frontend

**User Story:** As a developer, I want the frontend to build, so that the Vue application compiles without errors after the rebrand.

#### Acceptance Criteria

1. WHEN `npm run build` is executed in the `frontend` directory, THE Build_System SHALL terminate with a zero (success) exit code.
2. WHEN `npm run build` completes successfully in the `frontend` directory, THE Build_System SHALL produce the frontend build output artifacts.
3. IF the frontend build encounters one or more compilation errors, THEN THE Build_System SHALL terminate with a non-zero exit code and emit an error indication identifying the failing source file without producing the frontend build output artifacts.

### Requirement 14: Compile the backend

**User Story:** As a developer, I want the Go backend to compile, so that the migrated code is valid after package and import changes.

#### Acceptance Criteria

1. WHEN `go build ./...` is executed at the project root, THE Build_System SHALL compile every Go package in the HVRIns module and terminate with a success (zero) exit code.
2. IF the backend compilation encounters one or more errors, THEN THE Build_System SHALL terminate with a non-zero exit code, identify each failing package and source file, and leave the source files unmodified.

### Requirement 15: Produce the executable

**User Story:** As a developer, I want a full Wails build, so that a runnable HVRIns executable is produced.

#### Acceptance Criteria

1. WHEN `wails build` is executed at the project root, THE Build_System SHALL produce the executable file named `HVRIns.exe` in the `build/bin/` directory and SHALL terminate with a success (zero) exit status.
2. IF `wails build` terminates with a non-zero exit status, THEN THE Build_System SHALL report an error indicating the build failure and SHALL NOT report success for the absent or incomplete `HVRIns.exe` in the `build/bin/` directory.
3. WHEN the produced `build/bin/HVRIns.exe` is launched on Windows, THE HVRIns SHALL start and display its main application window without a runtime error dialog.

### Requirement 16: Visual verification

**User Story:** As a developer, I want to verify the rebrand visually, so that all branded elements render as intended in the running app.

#### Acceptance Criteria

1. WHILE HVRIns is running via `wails dev`, THE HVRIns SHALL display the application icon rendered with the Instagram_Palette gradient and the title bar Instagram camera logo.
2. WHILE HVRIns is running via `wails dev`, THE HVRIns SHALL display the sidebar with the `--sidebar-bg` background (`#0e0c18`) and a vertical Instagram_Palette gradient right border, and SHALL display the single-line Status_Bar footer.
3. WHILE HVRIns is running via `wails dev`, THE HVRIns SHALL render the action buttons with the `--brand-gradient` gradient (`linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D)`) and SHALL render accent elements with the Accent_Token `--accent` (`#E1306C`) with no leftover hard-coded blue or cyan literals.
4. WHEN the user navigates to the Accounts page, THE Status_Bar SHALL display the Accounts stats content within the `#status-bar-page-slot` Page_Slot element.
5. WHEN the user navigates away from the Accounts page, THE Status_Bar SHALL display only the CPU usage value, the RAM usage value, and the two icon-only utility buttons, with an empty `#status-bar-page-slot` Page_Slot.
