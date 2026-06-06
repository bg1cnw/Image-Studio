# 源码构建与验证

本文档聚焦源码构建、开发模式、验证脚本和 CI。预编译安装包与下载渠道见 [packages.md](./packages.md)。

## 环境要求

- Go 1.25.x。当前 `go.mod` 使用 `go 1.25.5` 与 `toolchain go1.26.3`。
- Node.js 20 或更新版本。
- Wails CLI v2.12.0。非 macOS release workflow 使用 `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`。
- Android 构建需要 JDK 17、Android SDK 34、Build Tools 34.0.0、Gradle 8.7。

## 克隆源码

```bash
git clone https://github.com/RoseKhlifa/Image-Studio.git
cd Image-Studio
```

如果你正在跟随当前开发分支而不是上游正式 release，建议改用你实际工作的 fork 地址。

## 桌面开发模式

```bash
cd image-studio
wails dev
```

`image-studio/wails.json` 当前会执行：

- `frontend:install`: `npm ci`
- `frontend:build`: `npm run build`
- `frontend:dev:watcher`: `npm run dev`

前端脚本会按宿主平台自动选择 `macos` / `windows` / `linux` 对应主题，不需要手动改环境变量。

## 前端独立预览

```bash
cd image-studio/frontend
npm ci

npm run dev
npm run dev:macos
npm run dev:windows
npm run dev:linux
npm run dev:android
npm run dev:android-pad
```

打包静态资源：

```bash
npm run build
npm run build:macos
npm run build:windows
npm run build:linux
npm run build:android
npm run build:android-pad
```

这些命令只切换 `VITE_TARGET_PLATFORM` 对应的主题和壳层，不改变主业务逻辑。

## macOS 本地发布包

```bash
bash scripts/package-local-macos-app.sh
```

产物位于：

```text
image-studio/build/bin/Image Studio.app
```

脚本会分别构建 arm64 与 amd64，再用 `lipo` 合成 universal 二进制，并执行本地自签。

## Windows / Linux Wails 构建

Wails v2 桌面端需要在目标平台原生构建。

Windows：

```bash
cd image-studio
wails build -platform windows/amd64 -clean
```

Linux Ubuntu 24.04 / Debian 新版本：

```bash
sudo apt-get update
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev

cd image-studio
wails build -platform linux/amd64 -clean -tags webkit2_41
```

Ubuntu 22.04 系通常使用 `libwebkit2gtk-4.0-dev`，构建时不加 `webkit2_41` tag。

## Windows / Linux Gio 测试客户端

Gio 客户端位于 `gio-client/`，与 Wails / WebView2 主实现独立。它复用 `go-cli/pkg/client` 请求内核，不读取 `image-studio/frontend/dist`。

Windows：

```bash
cd gio-client
go test ./...
go build -o ../dist/image-studio-gio.exe ./cmd/image-studio-gio
```

Linux：

```bash
sudo apt-get update
sudo apt-get install -y \
  pkg-config \
  libegl1-mesa-dev \
  libvulkan-dev \
  libwayland-dev \
  libx11-dev \
  libx11-xcb-dev \
  libxcursor-dev \
  libxfixes-dev \
  libxkbcommon-dev \
  libxkbcommon-x11-dev

cd gio-client
go test ./...
go build -o ../dist/image-studio-gio ./cmd/image-studio-gio
```

release workflow 会单独上传 `image-studio-gio-*` artifacts，不改变现有 `image-studio-*` Wails artifacts。

## Android APK

```bash
cd android-shell
./gradlew assembleRelease
```

Gradle 会先执行 `image-studio/frontend` 的 `npm run build:android`，再把 `dist/` 拷贝进 APK assets。APK 内部运行同一个 Android 前端目标，phone / pad 布局由运行时窗口尺寸和方向决定。

可选环境变量：

| 变量 | 用途 |
|---|---|
| `IMAGE_STUDIO_ANDROID_VERSION_NAME` | Android `versionName`。 |
| `IMAGE_STUDIO_ANDROID_VERSION_CODE` | Android `versionCode`。 |
| `IMAGE_STUDIO_KEYSTORE_PATH` | release 签名 keystore。未提供时使用自动生成的 debug keystore。 |
| `IMAGE_STUDIO_KEYSTORE_PASSWORD` | keystore 密码。 |
| `IMAGE_STUDIO_KEY_ALIAS` | key alias。 |
| `IMAGE_STUDIO_KEY_PASSWORD` | key 密码。 |
| `IMAGE_STUDIO_ANDROID_USE_PREBUILT_FRONTEND` | 设为 `1` / `true` 时复用已有 `frontend/dist`。 |

MuMu 模拟器调试流程见 [mumu-android-debug.md](./mumu-android-debug.md)。

## 版本元数据

release workflow 会先执行：

```bash
./scripts/compute-version.sh
```

它会从 tag 或 `image-studio/wails.json` 计算桌面版本、前端版本、Android `versionName` / `versionCode`。随后 `scripts/sync-version-metadata.mjs` 会同步：

- `image-studio/wails.json`
- `image-studio/frontend/package.json`
- `image-studio/frontend/package-lock.json`

本地不要手动维护多份版本号，除非你明确在调整下一次发布的基准版本。

## 验证入口

常用验证：

```bash
cd image-studio/frontend
npm run test
npm run build

cd ../..
cd image-studio
GOPATH="../.gopath" GOMODCACHE="../.gomodcache" GOCACHE="../.gocache" go test ./...

cd ../go-cli
GOPATH="../.gopath" GOMODCACHE="../.gomodcache" GOCACHE="../.gocache" go test ./...

cd ../gio-client
GOPATH="../.gopath" GOMODCACHE="../.gomodcache" GOCACHE="../.gocache" go test ./...
```

跨平台内核本地全量验证：

```bash
node scripts/verify-local-platform-kernel.mjs
```

该脚本会跑前端测试 / 构建、Worker 测试、本地 smoke、Android debug assemble、Go 测试和 macOS 发布包验证。是否能全绿取决于本机 Android SDK / JDK 与 macOS 构建工具是否齐全。

其他入口：

```bash
node scripts/verify-local-macos-release.mjs
node scripts/local-smoke-check.mjs
node scripts/live-verify.mjs
```

真实上游对比验证需要先按 `scripts/live-verify.env.example` 准备 `.env.live` 或 `.env.local`。

## CI

当前发布链路在 `.github/workflows/release.yml`：

- 并行构建 Windows、macOS、Linux Wails 桌面产物。
- Windows 额外产出单个自适应架构的 NSIS installer `image-studio-<version>-windows-installer.exe`，内部同时包含 amd64 与 arm64 二进制，供正式安装分发或 Microsoft Store Win32 提交使用。
- 单独构建一个 Android release APK。
- tag 为 `v*` 时将所有产物附加到 GitHub Release。

### Windows 签名

Windows 11 的 Smart App Control / SmartScreen 会重点拦截无法验证发布者的 `exe`。仓库里的 release workflow 现在约定：

- 如果配置了 Windows 签名证书，workflow 会自动对 Windows `exe` 做 Authenticode 签名并校验。
- 如果没有配置签名证书，workflow 仍然继续产出 Windows artifact，但会在日志里明确警告这些 `exe` 可能被 Win11 拦截。

需要配置的 GitHub Actions secrets：

| 变量 | 用途 |
|---|---|
| `IMAGE_STUDIO_WINDOWS_CERT_BASE64` | Base64 编码后的 `.pfx` 证书内容。 |
| `IMAGE_STUDIO_WINDOWS_CERT_PASSWORD` | `.pfx` 密码。 |

可选 GitHub Actions variable：

| 变量 | 用途 |
|---|---|
| `IMAGE_STUDIO_WINDOWS_TIMESTAMP_URL` | RFC 3161 时间戳地址；默认 `http://timestamp.acs.microsoft.com`。 |

签名步骤由 `scripts/sign-windows-binary.ps1` 执行，内部会调用 `signtool sign`、`signtool verify /pa /all` 和 `Get-AuthenticodeSignature` 做校验。

平台内核验证 workflow：

- `.github/workflows/verify-platform-kernel.yml`
- `.github/workflows/live-verify-platform-kernel.yml`
