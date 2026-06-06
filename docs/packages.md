# 安装包与下载渠道

本文档说明预编译安装包的下载入口、命名规则和平台差异。源码构建见 [build.md](./build.md)，应用展示见 [showcase.md](./showcase.md)。

## 下载渠道

### 稳定版本

稳定版本到 [RoseKhlifa/Image-Studio Releases](https://github.com/RoseKhlifa/Image-Studio/releases) 下载，适合日常使用与对外分发。

### 抢先测试版本

如果你想体验当前分支最近完成、但还没有进入上游 tag 的改动，可以到 [DR-lin-eng/Image-Studio Actions · release.yml](https://github.com/DR-lin-eng/Image-Studio/actions/workflows/release.yml) 下载最近一次成功构建的 artifact。

这类构建通常会更快包含当前开发分支上的改动，但稳定性和回归覆盖不如正式 release，建议只在测试环境使用。

Windows 用户需要额外注意：Actions 里的 CI artifact 如果没有经过 Authenticode 签名，在 Windows 11 上很容易被 Smart App Control 或 SmartScreen 直接拦截，提示“无法确认其编写人”或“可能不安全的应用”。这类包不应作为对外分发的正式安装包。

## 安装包命名规则

当前 release workflow 会产出带版本号前缀的安装包，格式大致如下：

| 平台 | 文件名模式 | 说明 |
|---|---|---|
| Windows x64 | `image-studio-<version>-windows-amd64.exe` | 标准 Wails 桌面版，依赖 WebView2 Runtime。 |
| Windows ARM64 | `image-studio-<version>-windows-arm64.exe` | 面向 Windows on Arm 设备。 |
| macOS universal | `image-studio-<version>-macos-universal.zip` | 解压后得到 `Image Studio.app`。 |
| Linux x64 | `image-studio-<version>-linux-amd64.tar.gz` | 标准 Wails 桌面版。 |
| Linux ARM64 | `image-studio-<version>-linux-arm64.tar.gz` | 面向 ARM64 Linux 桌面环境。 |
| Android | `image-studio-<version>-android-release.apk` | 单 APK，运行时自适应 phone / pad 布局。 |
| Gio Windows | `image-studio-gio-<version>-windows-*.exe` | Gio 原生 GUI 测试版，不依赖 WebView2。 |
| Gio Linux | `image-studio-gio-<version>-linux-*.tar.gz` | Gio 原生 GUI 测试版，不依赖 WebKitGTK。 |

## 各平台选择建议

| 场景 | 建议下载 |
|---|---|
| 普通桌面用户 | 对应平台的 `image-studio-<version>-...` Wails 版。 |
| Windows 上没有稳定 WebView2 环境，或想验证 Gio 原生界面 | `image-studio-gio-<version>-windows-...`。 |
| Linux 上只想试 Gio 原生渲染路径 | `image-studio-gio-<version>-linux-...`。 |
| 手机与平板统一安装 | `image-studio-<version>-android-release.apk`。 |

## 平台注意事项

### Windows

- Wails 桌面版依赖 WebView2 Runtime；Windows 10+ 通常已预装。
- ARM64 设备优先下载 `windows-arm64`，避免 x64 仿真带来的额外开销。
- 对外分发请优先使用带有效 Authenticode 签名的正式 release；未签名的 CI `exe` 在 Windows 11 上可能被智能应用控制直接阻止运行。

### macOS

- 安装包是 universal 产物，同时覆盖 Apple Silicon 和 Intel。
- 如果被 Gatekeeper 拦截，可执行：

```bash
xattr -dr com.apple.quarantine "Image Studio.app"
```

或者在 Finder 中右键选择“打开”。

### Linux

- 预编译包主要面向带 GTK / WebKitGTK 依赖的桌面环境。
- 如果你更想避免 WebKitGTK 依赖，可以优先试 Gio 测试版。

### Android

- 当前只维护一个 `android-release.apk`，运行时根据窗口尺寸和方向切换 phone / pad 布局。
- Android 壳层会复用前端远程内核和本地桥接能力，不再分别维护两套 APK。

## 相关文档

- [build.md](./build.md)：源码构建、验证脚本、CI。
- [usage.md](./usage.md)：首次配置、API 形态选择、参数策略。
- [showcase.md](./showcase.md)：界面展示与能力概览。
