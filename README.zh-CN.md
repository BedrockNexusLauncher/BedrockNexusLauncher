# Bedrock Nexus

一个面向 **Windows** 的 **Minecraft Bedrock Edition (GDK)** 桌面启动器。

Bedrock Nexus 用于帮助你安装、管理、隔离、扩展并启动正式版或预览版环境，同时让内容与工具链更容易整理。

## 文档

- **用户文档：** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/
- **简体中文：** https://bedrocknexuslauncher.github.io/BedrockNexusLauncher/zh-CN/

## 功能

- **实例管理** — 并排安装、隔离并运行多个正式版与预览版实例，每个实例拥有独立的内容与设置。
- **一键启动** — 启动器在启动时会检查 Microsoft Gaming Services、Microsoft GameInput、Visual C++ 运行时与 WebView2 运行时，并可为你安装 GameInput 与 Visual C++ 运行时；若缺少 Gaming Services，需要从 Microsoft Store 安装。
- **内容管理** — 导入、整理、在实例之间转移并删除世界、资源包、行为包与皮肤包，并可将世界备份为 `.mcworld` 文件；将 `.mcpack`、`.mcaddon`、`.mcworld` 文件拖放到内容页面（或将 `.zip`/`.dll` 模组拖放到模组页面）即可导入。
- **模组支持** — 一流的 [LeviLamina](https://github.com/LiteLDev/LeviLamina) 支持（按实例安装、更新与版本锁定），以及 [lip](https://github.com/LiteLDev/lip) 包浏览与管理。
- **CurseForge 集成** — 直接从 CurseForge 搜索并安装 Bedrock 模组。
- **社区市场** — 从 GitHub 托管的社区索引浏览并安装包。
- **世界工具** — 内置 `level.dat` 编辑器与按版本划分的截图库。
- **下载管理器** — 支持断点续传、自动重试与实时进度的下载。
- **多语言** — 12 种语言，翻译完整度一致，并为波斯语提供从右到左的文本支持。
- **自动更新** — 检查 GitHub Releases 并可下载、安装更新（在发布版本后生效）。

## 下载地址

目前尚未发布公开版本，因此暂无可下载的安装包。你可以按照下方的开发说明从源码构建。安装包将发布在 **GitHub Releases：** https://github.com/BedrockNexusLauncher/BedrockNexusLauncher/releases

## 使用要求

- Windows 10/11
- 拥有 Minecraft Bedrock Edition (GDK) 的正版授权
- 系统中可用的 Microsoft Gaming Services 与 Microsoft GameInput

## 社区

- **Discord：** https://discord.gg/v5R5P4vRZk
- **QQ 群：** https://qm.qq.com/q/1z791rJgJG

## 问题反馈

- **Issues：** https://github.com/BedrockNexusLauncher/BedrockNexusLauncher/issues
- 提交时建议附带 Windows 版本、Bedrock Nexus 版本、复现步骤，以及相关日志或截图。

## 文档本地开发

用户文档现已迁移到 GitHub Pages。若你要在本地预览文档站：

```bash
cd docs
npm install
npm run docs:dev
```

## 开发

Bedrock Nexus 的后端使用 [Go](https://go.dev) + [Wails v3](https://v3alpha.wails.io)，前端使用 React 19 + TypeScript + Tailwind CSS 4 + HeroUI。

前置要求：

- [Go](https://go.dev/dl/) 1.25.3 或更高版本
- [Node.js](https://nodejs.org) 20 或更高版本（npm）
- [Wails v3 CLI](https://v3alpha.wails.io/reference/cli)：`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- [Task](https://taskfile.dev)：`go install github.com/go-task/task/v3/cmd/task@latest`

```bash
# 安装前端依赖
cd frontend && npm install && cd ..

# 开发模式（热重载）
wails3 task dev

# 生产构建
wails3 task build

# 打包安装程序（Inno Setup）
wails3 task package:inno

# 类型检查与前端验证
wails3 task verify
```

## 参与贡献

欢迎提交 Pull Request！对于较大的改动，请先开 Issue 讨论，并遵循现有的代码风格（后端位于 `internal/`，前端位于 `frontend/src/`）。

## 许可证

本项目使用 **GPL-3.0-only** 许可证，完整文本见 `COPYING`。Bedrock Nexus 是 LiteLDev/LeviLauncher（GPL-3.0-only）的分支；第三方组件见 `THIRD_PARTY_NOTICES`。

## 免责声明

Bedrock Nexus 是一个非官方的社区启动器，与 Mojang Studios 或 Microsoft 没有任何关联、认可或合作关系。使用它需要拥有正版授权的 Minecraft Bedrock Edition。"Minecraft" 是 Mojang Studios 的商标。
