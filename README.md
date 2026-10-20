<p align="center">
  <img src="assets/bedrock-nexus-logo.png" width="160" alt="Bedrock Nexus logo" />
</p>

<h1 align="center">Bedrock Nexus</h1>

<p align="center">
  A modern, free and open-source desktop launcher for <b>Minecraft Bedrock Edition (GDK)</b> on Windows.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/platform-Windows%2010%2F11-blue" alt="Platform" />
  <img src="https://img.shields.io/badge/license-GPL--3.0-green" alt="License" />
</p>

---

## ✨ Features

- **Instance management** — install, isolate and run multiple Release and Preview builds side by side, each with its own content and settings.
- **One-click launch** — at startup the launcher checks Microsoft Gaming Services, Microsoft GameInput, the Visual C++ runtime and the WebView2 runtime, and can install GameInput and the Visual C++ runtime for you. If Gaming Services is missing, you'll be asked to install it from the Microsoft Store.
- **Content management** — import, organize, transfer between instances and delete worlds, resource packs, behavior packs and skin packs, and back up worlds to `.mcworld` files; drag-and-drop `.mcpack`, `.mcaddon` and `.mcworld` files onto the Content page (or `.zip`/`.dll` mods onto the Mods page) to import them.
- **Modding** — first-class [LeviLamina](https://github.com/LiteLDev/LeviLamina) support with per-instance installation, updates and version pinning, plus [lip](https://github.com/LiteLDev/lip) package browsing and management.
- **CurseForge integration** — search and install Bedrock mods directly from CurseForge.
- **Community marketplace** — browse and install packs from a GitHub-hosted community index.
- **World tools** — a built-in `level.dat` editor for world properties, and a per-version screenshot gallery.
- **Download manager** — resumable downloads with automatic retries and live progress for versions, mods and runtime dependencies.
- **Beautiful and personal** — light/dark themes, custom accent colors, wallpapers, optional themed strokes, and smooth animations with a disable-animations kill switch.
- **Truly multilingual** — 12 locales (English, 简体中文, 繁體中文, Русский, 日本語, Deutsch, Español, Français, Italiano, 한국어, Português, فارسی) with full translation parity, and right-to-left text support for Persian (RTL text alignment, mirrored settings layout and a Persian-friendly font stack).
- **Discord Rich Presence** — show what you're playing while the game runs.
- **Automatic updates** — the launcher checks GitHub Releases for new versions and can download and install an update, restarting itself to finish. (Active once releases are published.)
- **Built-in diagnostics & bug reporting** — generate a diagnostics report (app and OS version, architecture, and a startup-log excerpt with your user-profile path redacted) and open pre-filled GitHub issues from inside the app.

## 📥 Downloads

No public release has been published yet, so there is currently no installer to download. You can build Bedrock Nexus from source — see [Development](#️-development) below. Installers will be published on [GitHub Releases](https://github.com/BedrockNexusLauncher/BedrockNexusLauncher/releases).

## 📋 Requirements

- Windows 10 or 11 (x64)
- A licensed copy of Minecraft Bedrock Edition (GDK) from the Microsoft Store
- Microsoft Gaming Services and Microsoft GameInput — the launcher checks for these at startup; it can set up GameInput (and the Visual C++ runtime) automatically, while Gaming Services must be installed from the Microsoft Store if it is missing

## 📖 Documentation

- **English:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/
- **简体中文:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/zh-CN/
- **繁體中文:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/zh-HK/
- **Español:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/es-ES/
- **Русский:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/ru-RU/
- **Deutsch:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/de-DE/
- **日本語:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/ja-JP/
- **한국어:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/ko-KR/
- **Français:** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/fr-FR/

## 💬 Community

- **Discord:** https://discord.gg/v5R5P4vRZk
- **QQ Group:** https://qm.qq.com/q/1z791rJgJG

## 🐛 Reporting Issues

Found a bug or have a feature idea? Please [open an issue](https://github.com/BedrockNexusLauncher/BedrockNexusLauncher/issues).
The quickest way is **Settings → Report a Problem** (or **About → Report a Problem**) in the launcher — it attaches a diagnostics report (with your user-profile path redacted) automatically.

## 🛠️ Development

Bedrock Nexus is built with [Go](https://go.dev) + [Wails v3](https://v3alpha.wails.io) on the backend and React 19 + TypeScript + Tailwind CSS 4 + HeroUI on the frontend.

### Prerequisites

- [Go](https://go.dev/dl/) 1.25.3 or newer
- [Node.js](https://nodejs.org) 20 or newer (npm)
- [Wails v3 CLI](https://v3alpha.wails.io/reference/cli): `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- [Task](https://taskfile.dev): `go install github.com/go-task/task/v3/cmd/task@latest`

### Run & build

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Start the app in development mode (hot reload)
wails3 task dev

# Production build
wails3 task build

# Package an installer (Inno Setup)
wails3 task package:inno

# Type-check and validate the frontend
wails3 task verify
```

### Documentation site

The user guide is a VitePress site in [`docs/`](docs/):

```bash
cd docs
npm install
npm run docs:dev
```

## 🤝 Contributing

Pull requests are welcome! For substantial changes, please open an issue first to discuss what you would like to change, and follow the existing code style (the backend is grouped under `internal/`, the frontend under `frontend/src/`).

## 📄 License

This project is licensed under the **GNU General Public License v3.0 only (GPL-3.0-only)** — see [COPYING](COPYING) for the full text.

Bedrock Nexus is a fork of [LiteLDev/LeviLauncher](https://github.com/LiteLDev/LeviLauncher), which is also licensed under GPL-3.0-only. Bundled components are listed in [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES) and remain under their own terms where stated there.

## ⚠️ Disclaimer

Bedrock Nexus is an unofficial community launcher and is not affiliated with, endorsed by, or associated with Mojang Studios or Microsoft. You must own a legitimate licensed copy of Minecraft Bedrock Edition to use it. "Minecraft" is a trademark of Mojang Studios.
