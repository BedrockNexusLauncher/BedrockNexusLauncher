# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-01

First public release of Bedrock Nexus — a modern, free and open-source desktop launcher for Minecraft Bedrock Edition (GDK) on Windows, built with Go + Wails v3 and React 19 + TypeScript + Tailwind CSS 4 + HeroUI.

### Added

**Launcher core**

- Desktop launcher app for Windows 10/11 with a custom WebView2-based UI, frameless window with native min/max/close controls and single-instance guard.
- Installation and side-by-side management of isolated Minecraft Bedrock instances (Release and Preview), each with its own content, settings and version registration.
- One-click game launch with automatic dependency checks and setup for Microsoft Gaming Services, Microsoft GameInput and the Visual C++ runtime, plus a WebView2 runtime startup check.
- Version registration/unregistration in Windows so instances appear in the system app list and can be launched directly.
- Automatic application updates from GitHub Releases with an in-app update flow and restart handling.
- Discord Rich Presence for launcher and game status.
- Startup diagnostics written to a local log, with sanitized in-app diagnostics reports for bug reports.

**Content management**

- Import, browse, organize, export and delete worlds, resource packs, behavior packs and skin packs.
- Drag-and-drop import for `.mcpack`, `.mcaddon`, `.mcworld`, `.mcpack` skin packs and archives anywhere in the window, with per-player target selection and duplicate/overwrite resolution.
- Per-version screenshot gallery with preview and deletion.
- World `level.dat` editor (game mode, difficulty, hardcore and other world properties).
- Resource pack `.material.bin` compatibility checking and updates.
- Cross-version transfer of packs and worlds between isolated instances.

**Modding**

- First-class LeviLamina support: per-instance install, update, uninstall and exact game-version compatibility mapping.
- Integrated [lip](https://github.com/LiteLDev/lip) package browsing, search, install and cache management via an embedded daemon.
- CurseForge mod search and installation with mod compatibility intelligence.
- Community marketplace backed by a GitHub-hosted index (`index.json` + Releases) with search, filters, screenshots and a guided install flow; anyone can publish via the documented upload script.
- Experimental MCPEDL catalogue browser (behind a feature flag).

**Downloads**

- Resumable, throttled HTTP download manager with persistent validators, retries, live progress events and a dedicated downloads page.

**User interface & experience**

- Light/dark themes, custom accent colors, background wallpapers and base colors, optional themed strokes, ambient glow and a disable-animations kill switch.
- Responsive layout with a sidebar mode and a navbar mode.
- First-launch onboarding wizard (content paths, language) and a graphical elevation-consent screen for the startup patch.
- Settings pages covering appearance, language, paths, Discord RPC, startup patch behavior, experimental features, cache management and bug reporting.
- Toast notifications, modals and pre-filled GitHub bug reports with optional diagnostics.

**Internationalization**

- 12 locales: English, Simplified Chinese, Traditional Chinese (Hong Kong), Russian, Japanese, German, Spanish, French, Italian, Korean, Portuguese and Persian.
- Full right-to-left support for Persian: engine layout mirroring (auto/LTR/RTL preference), RTL text alignment rules, Persian-friendly font stack and a locked-RTL paragraph style for mixed Persian/Latin sentences.

**Infrastructure**

- CI workflow building the Windows executable and Inno Setup installer on every push, pull request and version tag.
- VitePress documentation site (English and Simplified Chinese) published to GitHub Pages, with additional localized guides.
- Locale maintenance scripts for missing/unused key detection and cross-file comparison.

[1.0.0]: https://github.com/BedrockNexusLauncher/BedrockNexusLauncher/releases/tag/v1.0.0
