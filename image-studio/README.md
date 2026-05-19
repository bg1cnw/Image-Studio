# Image Studio

> 一个开源的 GPTCODEX 图像生成 / 编辑桌面客户端 ·
> **流式 SSE 保活,兼容 Cloudflare 524/504 超时截断** ·
> Wails(Go + React/TS)

![license](https://img.shields.io/badge/license-MIT-7c3aed)
![go](https://img.shields.io/badge/go-%3E%3D1.25-00ADD8)
![react](https://img.shields.io/badge/react-18-61DAFB)
![wails](https://img.shields.io/badge/wails-v2-DF0000)
![platform](https://img.shields.io/badge/platform-windows%20%7C%20macOS%20%7C%20linux-lightgrey)

---

## 为什么写这个

大多数 OpenAI/GPTCODEX 中转站把图像生成接口架在 Cloudflare 后面,而 image-2 这种模型推理需要 30~120 秒。一旦超过 CF 的 100s 默认网关超时,连接被切断 → 客户端收到 524/504 → 整张图作废。

常见后果:你点了「生成」,等 100 秒,然后看到 `Cloudflare Error code 524 · A timeout occurred`,再等几分钟再试,大概率重复。

**本项目的核心解法是用上游的 `/v1/responses` 接口(类似 OpenAI Responses API)+ SSE 流式协议**:
- 模型边推理边发心跳事件(`response.in_progress` / `image_generation_call.generating` / `partial_image` ...),保持 TCP 连接活跃,CF 看到流量就不会断
- 即使真的被掐了,本地有 3 次自动重试 + 15 秒 backoff
- 收到 `partial_image_b64` 也会兜底保存,作为「半成品」给你看效果

把这个 Python CLI 脚本(`generate_gptcodex_image.py`)重写为 Go,再封装成 Wails 桌面应用,补全了画板、蒙版、多参考图、撤销重做、历史对比、多标签页等图像编辑器该有的能力。

---

## 核心亮点

| | |
|---|---|
| 🛡 **SSE 流式保活** | 用 `/v1/responses` 接口而非 `/v1/images/generations`,以 Server-Sent Events 持续从上游接收事件,Cloudflare/Nginx 网关看到持续流量不会判定为空闲超时 |
| 🔁 **自动重试 + 部分结果回退** | 识别 Cloudflare 524/504、JSON `status` 5xx、`retryable=true` 自动重试 3 次;final result 没拿到时用最后的 `partial_image_b64` 兜底 |
| 📦 **大 base64 单行缓冲** | `partial_image_b64` 单行可超 4MB,自定义 8MB scanner buffer,不会被 Go bufio 默认的 64KB 截断 |
| 🔌 **双 transport** | 默认 net/http 直连;遇到 TLS 指纹 / 代理问题可一键切到子进程 `curl`(`--http1.1 --ssl-no-revoke`) |
| ⏹ **真正可取消** | `context.Context` 端到端,包括 `exec.CommandContext` 给 curl,「取消」按钮能立刻中断 in-flight 请求 |
| 🖼 **完整图像编辑器** | Konva 画布、蒙版+橡皮、4 种标注、旋转/翻转/裁剪、历史对比、并发批量(1/2/4/8) |
| 🧩 **多标签 workspace** | 浏览器风的多 tab,每个独立 prompt/参数/源图;切换不丢现场 |
| 💾 **100% 本地数据** | 无遥测、无云端账户、无内购;API key、历史、生成图都在你的机器上 |

---

## 截图

(可在 [Releases](https://github.com/RoseKhlifa/Image-Studio/releases) 页面查看运行截图,或自行 build 一份试试。)

---

## 安装

### 方式 1:下载预编译版本(推荐)

去 [Releases](https://github.com/RoseKhlifa/Image-Studio/releases) 下载 `image-studio.exe`(Windows 单文件,约 12MB),双击即可运行。Windows 10+ 需要预装 [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/)(大部分新机器已预装)。

### 方式 2:从源码构建

```bash
# 环境要求
#  - Go >= 1.25
#  - Node >= 20
#  - Wails CLI v2:  go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/RoseKhlifa/Image-Studio.git
cd image-studio/image-studio

# 开发模式(Vite 热重载 + DevTools)
wails dev

# 生产构建,输出到 build/bin/image-studio.exe (~12MB)
wails build
```

支持 macOS 和 Linux(`wails build -platform darwin/amd64` / `linux/amd64`),代码层面跨平台,但作者只在 Windows 11 上日常用。

---

## 快速上手

1. 打开应用,点击左侧 **「API Key」** 折叠区,粘贴你的 GPTCODEX key(`sk-...`)
2. ⚠️ **重要**:在中转站后台把这个 key 绑定到**拥有 `gpt-5.5` 模型的分组**(余额分组 / 套餐分组),**不要** 选 image-2 分组。详见应用内 FAQ 或下方 [API Key 配置](#api-key-配置)
3. (可选)点击 **「🔌 测试连接」** 按钮验证 key 是否有 gpt-5.5 权限
4. 输入 prompt(可在右上角 📋 选模板/历史),选风格 / 比例 / 质量 / 数量
5. 按 `Ctrl + Enter` 或点击 **「生成 N 张」**

图生图流程:
- 拖入本地图片到窗口 / Ctrl+V 粘贴 / 「+ 添加图片」按钮 → 累积参考图列表
- 切到「图生图」模式 → 写修改要求 → 生成

---

## API Key 配置

**本应用调用的是上游的 `/v1/responses` 接口**(OpenAI Responses API 形态),而不是直接的 `/v1/images/generations`。图像生成是通过模型内置的 `image_generation` 工具触发的。

所以你的 key 必须:
- ✅ 绑定到**拥有 `gpt-5.5` 模型的分组**(中转站后台叫做「余额分组」或「套餐分组」)
- ❌ **不要**选「image-2 分组」 — 那是直接 image API 的分组,不包含 gpt-5.5,会返回 401/403 或 `model not found`

更换上游 / 模型:左侧设置面板可改 `BASE_URL`、文本模型 ID(默认 `gpt-5.5`)、图像模型 ID(默认 `gpt-image-2`)。任何兼容 OpenAI **Responses API** 形态 + 提供 `image_generation` 工具的中转站理论上都行;**不兼容**普通 `/v1/chat/completions` 中转站。

---

## 完整功能

### 生成
- 文生图 / 图生图(支持多张参考图,可拖动重排顺序)
- 输入图源:文件对话框 / 拖拽窗口 / Ctrl+V 粘贴 / 从历史复用 / 双击历史项
- 参数:5 种比例 · 3 档质量(1K/2K/4K)· seed · negative prompt · 5 种风格 chip
- **并发批量** 1 / 2 / 4 / 8 张,每张自动随机 seed
- 上游可配:BASE_URL、文本模型 ID、图像模型 ID、传输通道(native/curl)
- prompt 历史(自动去重,cap 50)+ 8 个内置模板(写实/二次元/水彩/像素等)

### 画板(Konva)
- 缩放(鼠标滚轮以指针为中心)/ 拖动 / 双击 fit ↔ 100%
- **蒙版**:画笔 + 橡皮,大小滑块,实时半透明叠加
- **标注**:矩形 / 箭头 / 自由画笔 / 文字,8 色,选中后 Delete 删除
- **图变换**(后端 Go image 库):旋转 90°/-90°、水平/竖直翻转、矩形选区裁出
- **历史对比**:Shift+点击进入,左右分屏 + 中间可拖动 split bar
- 全屏 F11(隐藏左右栏专注画板)

### 历史
- IndexedDB 持久化(浏览器存储,无服务器)
- 搜索 prompt / 筛选 mode / 筛选日期(今天/本周/全部)
- **右键菜单**:复制 prompt / 复制本地路径 / 查看 raw 响应 / 设为源图 / 用作对比 / **以此参数重新生成** / **应用参数(不生成)**
- JSON 导入 / 导出(跨设备迁移)

### 工作流
- **多 workspace 标签页**:每个独立 prompt / 参数 / 源图 / 当前图;Ctrl+N 新建,Ctrl+W 关闭
- 切换 tab 自动按 prompt 前 18 字命名
- 撤销 / 重做统一 timeline:蒙版笔触、标注、清空都在栈里
- Ctrl+C 复制当前图到剪贴板 · Ctrl+V 粘贴
- 错误 banner 可关闭 + 「↻ 重试上次请求」
- **Toast** 通知(成功 / 错误 / 警告)+ **系统通知**(窗口失焦时生成完成弹 Windows toast)
- 进度估算(基于最近 5 次耗时滚动平均)

### 设置
- 主题:深色 / 浅色
- 字号:小 / 中 / 大
- 参数预设保存(尺寸 + 质量 + 风格 + 批量,常用配置一键应用)
- 历史导入 / 导出 JSON
- 清除 API Key / 清空历史
- 关于:版本号、MIT 协议链接、GitHub 仓库 / Issues 一键跳转

---

## 键盘快捷键

| 快捷键 | 功能 |
|---|---|
| `Ctrl` + `Enter` | 提交生成(textarea 内也能用) |
| `Ctrl` + `N` / `Ctrl` + `W` | 新建 / 关闭 workspace 标签 |
| `Ctrl` + `Z` / `Ctrl` + `Shift` + `Z` / `Ctrl` + `Y` | 撤销 / 重做 |
| `Ctrl` + `C` | 复制当前画板图到剪贴板 |
| `Ctrl` + `V` | 粘贴剪贴板图到画板(进入图生图) |
| `1` / `2` / `3` | 切换 拖动 / 蒙版 / 标注 工具 |
| `空格` | 按住临时切到拖动 |
| `F` | 重置视图(适配窗口) |
| `双击画板` | fit ↔ 100% 切换 |
| `F11` | 全屏模式 |
| `[` / `]` | 笔刷大小 −/+ 5 |
| `Esc` | 取消生成 → 退出对比 → 清除选中 → 关闭错误 |
| `Delete` | 删除选中的标注 |
| `Shift` + 点击历史 | 设为对比图 B |
| `双击` 历史 | 作为源图 |
| `右键` 历史 | 上下文菜单 |

---

## 数据存储位置

| 类型 | 位置 |
|---|---|
| API Key | 浏览器 `localStorage`(明文) |
| 历史记录元数据 | 浏览器 IndexedDB |
| 生成的 PNG | `%APPDATA%\image-studio\images\` |
| 拖入 / 变换的图 | `%APPDATA%\image-studio\imports\` |
| 原始 SSE 响应 | 跟 PNG 同目录(`gptcodex-response-*.txt`,排错用) |
| 用户偏好(主题、预设、prompt 历史) | `localStorage` |

数据完全不出本地;唯一的外部网络请求是向你配置的上游 BASE_URL 发起的生成请求本身。

---

## 故障排除

### 一直 524 / 504

- 上游网关超时在很多中转站很常见。本应用自动重试 3 次,如果都失败:
  - 检查 key 是否过期 / 余额 / 是否绑对了分组(见上方 [API Key 配置](#api-key-配置))
  - 设置 → 网络通道 改成 `curl`,有时能绕过原生 HTTP 的 TLS 指纹问题
  - 历史项 右键 → **查看 raw 响应**,看上游具体返回了什么(原始 SSE 全文)

### `model not found` / 401 / 403

Key 没有 `gpt-5.5` 模型权限。换分组,或者把图像模型 ID 改成你的 key 实际支持的(设置 → 图像模型 ID)。

### 多参考图 / 蒙版 / seed 没生效

这些字段在 payload 里都正确发送,**但上游是否真正使用取决于中转站和模型实现**。蒙版作为 tool 的 `mask` 字段、seed 作为 `seed` 字段、negative prompt 作为 `negative_prompt` 字段;不同模型实现对这些字段的支持程度不一。

完整 FAQ 请见应用内「API Key 区右上角的 ❓ FAQ」入口。

---

## 项目结构

```
.
├── go-cli/                       # 独立 Go CLI(可单独跑,共享 pkg/client)
│   ├── pkg/client/               # 核心:payload 构建 / SSE 解析 / 重试 / 双 transport
│   ├── internal/{promptui,fsio}/ # 终端交互 + 文件 IO
│   └── cmd/gptcodex-image/main.go
├── image-studio/                 # Wails 桌面应用
│   ├── backend/                  # Go bindings(按职责分文件)
│   │   ├── service.go            # Service + 生命周期 + Generate/Edit/Cancel
│   │   ├── types.go              # JSON-bound structs
│   │   ├── dialogs.go            # 文件对话框 / URL 跳转 / 历史导入导出
│   │   ├── imports.go            # 拖拽/粘贴 import + 文件名 sanitize
│   │   ├── imageops.go           # 旋转 / 翻转 / 裁剪(Go image 库)
│   │   ├── paths.go              # 目录解析 + 文件名构造
│   │   └── open.go               # 跨平台 OS 打开
│   ├── frontend/src/             # React + TS
│   │   ├── components/
│   │   │   ├── layout/           # AppHeader / WorkspaceBar / FooterBar
│   │   │   ├── panel/            # ControlPanel / SettingsPanel / PromptPopover / FAQModal
│   │   │   ├── canvas/           # CanvasStage / Toolbar / SourceStrip / StatusBar / EmptyState
│   │   │   ├── history/          # HistoryRail / RawResponseModal
│   │   │   └── common/           # Modal / ToastContainer / ContextMenu
│   │   ├── state/                # zustand store
│   │   ├── styles/               # 拆分的 CSS modules
│   │   ├── lib/                  # localStorage / idb-keyval 工具
│   │   └── types/                # 领域类型
│   └── build/bin/                # 生产 .exe 输出
└── go.work                       # Go workspace(backend replace ../go-cli)
```

---

## 致谢 / Stack

- [Wails v2](https://wails.io/) — Go 后端 + Web 前端的桌面应用框架
- [React 18](https://react.dev/) + [TypeScript](https://www.typescriptlang.org/) + [Vite](https://vitejs.dev/)
- [zustand](https://github.com/pmndrs/zustand) — 状态管理
- [react-konva](https://konvajs.org/docs/react/) — 画布渲染
- [idb-keyval](https://github.com/jakearchibald/idb-keyval) — IndexedDB 简易封装

灵感来自实际使用中频繁被中转站 524 截断的痛苦经历。

---

## License

[MIT](./LICENSE) © 2026

[![Star History Chart](https://api.star-history.com/svg?repos=RoseKhlifa/Image-Studio&type=Date)](https://star-history.com/#RoseKhlifa/Image-Studio&Date)
