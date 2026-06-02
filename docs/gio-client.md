# Gio 高性能测试客户端

Gio 客户端是 Windows / Linux 的独立测试版本，目录为 `gio-client/`。它的目标是验证不经过 WebView2/WebKitGTK 的原生 Gio 渲染路径，同时保持当前请求内核不变。

## 边界

- 不改 `image-studio/main.go`，不影响 Wails / WebView2 桌面端。
- 不改 `image-studio/frontend/` 的 React 视觉实现。
- 复用 `go-cli/pkg/client` 的 Responses API、Images API、SSE、retry、proxy、模型默认值和请求字段策略。
- Gio 前端为新的 immediate-mode 架构，UI 结构沿用桌面端的控制面板、画布、历史记录和运行日志布局。
- GUI 入口仅面向 Windows / Linux；其他平台只编译 unsupported stub，避免误判为 macOS 支持。

## WebView2 兼容状态

Gio 不直接解析 Chromium IndexedDB/LocalStorage 的内部文件，而是通过共享兼容状态文件与 WebView2 互通：

```text
<stable data root>/compat/state.json
```

该文件保存非敏感配置、profiles、active profile、prompt presets、prompt history、trusted output roots 和历史记录。API Key 不进入 JSON；两个客户端共用系统 keyring，service 为 `Image Studio`，user 为 `api-key:profile:<profile-id>`。

Windows 的 stable data root 与 WebView2 一致，来自 `HKCU\Software\YuanHua\Image Studio\DataRoot`，默认是 `Documents\Image Studio`。Linux 使用系统用户配置目录下的 `image-studio`。

互换规则：

- WebView2 启动时如果共享状态比本地 marker 更新，会导入配置和历史到现有 localStorage/IndexedDB。
- WebView2 运行中配置或历史变化后，会 debounce 写回共享状态。
- Gio 启动时读取同一共享状态，运行或关闭时写回配置，生成成功后追加历史。
- 共享状态解析失败时 Gio 不覆盖原文件，避免坏 JSON 导致历史丢失。

## 构建

```bash
cd gio-client
go test ./...
go build -o /tmp/image-studio-gio ./cmd/image-studio-gio
```

Linux 需要 Gio 原生依赖:

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
```

Release workflow 会额外产出 `image-studio-gio-*` artifacts。它们与现有 `image-studio-*` Wails artifacts 分开上传。
