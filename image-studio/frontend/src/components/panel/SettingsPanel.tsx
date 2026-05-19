import { useEffect, useState } from "react";
import { useStudioStore } from "../../state/studioStore";
import { GetOutputDir, OpenOutputDir, OpenExternalURL } from "../../../wailsjs/go/backend/Service";
import type { TransportKind } from "../../types/domain";
import { Modal } from "../common/Modal";

const REPO_URL = "https://github.com/RoseKhlifa/Image-Studio";
const MIT_URL = "https://opensource.org/licenses/MIT";

function PresetsRow() {
  const { presets, savePreset, applyPreset, deletePreset } = useStudioStore();
  function onSave() {
    const name = prompt("预设名:");
    if (name) savePreset(name);
  }
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {presets.map((p) => (
        <div key={p.id} style={{ display: "flex", gap: 4, alignItems: "center" }}>
          <button className="btn secondary" style={{ flex: 1, padding: "4px 8px", fontSize: 11 }} onClick={() => applyPreset(p.id)} title={`${p.size} · ${p.quality} · ×${p.batchCount}`}>
            {p.name}
          </button>
          <button className="tool-btn" onClick={() => deletePreset(p.id)} title="删除">✕</button>
        </div>
      ))}
      <button className="btn secondary" style={{ fontSize: 11, padding: "4px 8px" }} onClick={onSave}>
        + 保存当前参数
      </button>
    </div>
  );
}

export function SettingsPanel() {
  const {
    transport, baseURL, textModelID, imageModelID,
    theme, fontScale,
    setField, setAPIKey,
    history,
    exportHistory, importHistory,
    setTheme, setFontScale,
  } = useStudioStore();

  const [open, setOpen] = useState(false);
  const [outputDir, setOutputDir] = useState("");
  const [aboutOpen, setAboutOpen] = useState(false);

  useEffect(() => {
    if (!open) return;
    GetOutputDir().then(setOutputDir).catch(() => undefined);
  }, [open]);

  function clearAPIKey() {
    if (!confirm("确定清除已保存的 API Key 吗?")) return;
    setAPIKey("");
  }

  async function clearHistory() {
    if (!confirm(`确定清除 ${history.length} 条历史记录吗?(本地 IndexedDB 也会删除)`)) return;
    for (const h of history) {
      await useStudioStore.getState().deleteHistoryItem(h.id);
    }
  }

  return (
    <section className="settings-block">
      <button
        className="settings-toggle"
        onClick={() => setOpen((v) => !v)}
        title="高级设置"
      >
        <span>⚙ 设置</span>
        <span style={{ opacity: 0.5 }}>{open ? "▾" : "▸"}</span>
      </button>
      {open && (
        <div className="settings-body">
          <div className="settings-row">
            <label className="head">网络通道</label>
            <select
              className="select"
              value={transport}
              onChange={(e) => setField("transport", e.target.value as TransportKind)}
            >
              <option value="auto">auto(原生 HTTP)</option>
              <option value="native">native(强制原生)</option>
              <option value="curl">curl(子进程兜底)</option>
            </select>
            <small style={{ color: "var(--text-dim)", fontSize: 10 }}>
              如果遇到网络问题(Cloudflare TLS / 公司代理),试 curl
            </small>
          </div>

          <div className="settings-row">
            <label className="head">输出目录</label>
            <div className="source-pill">
              <span className="name" title={outputDir}>{outputDir || "..."}</span>
              <button onClick={() => OpenOutputDir().catch(() => undefined)} title="在资源管理器中打开">📂</button>
            </div>
          </div>

          <div className="settings-row">
            <label className="head">上游 BASE_URL(留空=默认 gptcodex.top)</label>
            <input
              className="input"
              type="text"
              value={baseURL}
              placeholder="https://gptcodex.top"
              onChange={(e) => setField("baseURL", e.target.value)}
              spellCheck={false}
            />
          </div>

          <div className="settings-row">
            <label className="head">文本模型 ID</label>
            <input
              className="input"
              type="text"
              value={textModelID}
              placeholder="留空=默认 gpt-5.5"
              onChange={(e) => setField("textModelID", e.target.value)}
              spellCheck={false}
            />
          </div>

          <div className="settings-row">
            <label className="head">图像模型 ID</label>
            <input
              className="input"
              type="text"
              value={imageModelID}
              placeholder="留空=默认 gpt-image-2"
              onChange={(e) => setField("imageModelID", e.target.value)}
              spellCheck={false}
            />
          </div>

          <div className="settings-row" style={{ flexDirection: "row", gap: 8 }}>
            <button className="btn secondary" onClick={clearAPIKey}>清除 API Key</button>
            <button className="btn secondary" onClick={clearHistory}>清空历史</button>
          </div>

          <div className="settings-row" style={{ flexDirection: "row", gap: 8 }}>
            <button className="btn secondary" onClick={exportHistory} style={{ flex: 1 }} title="导出全部历史为 JSON">
              📤 导出历史
            </button>
            <button className="btn secondary" onClick={importHistory} style={{ flex: 1 }} title="从 JSON 文件导入">
              📥 导入历史
            </button>
          </div>

          <div className="settings-row">
            <label className="head">主题</label>
            <div className="row" style={{ gap: 4 }}>
              <button
                className={`btn ${theme === "dark" ? "" : "secondary"}`}
                style={{ fontSize: 11, padding: "4px 10px", flex: 1 }}
                onClick={() => setTheme("dark")}
              >
                🌙 深色
              </button>
              <button
                className={`btn ${theme === "light" ? "" : "secondary"}`}
                style={{ fontSize: 11, padding: "4px 10px", flex: 1 }}
                onClick={() => setTheme("light")}
              >
                ☀ 浅色
              </button>
            </div>
          </div>

          <div className="settings-row">
            <label className="head">字号 {Math.round(fontScale * 100)}%</label>
            <div className="row" style={{ gap: 4 }}>
              {[0.85, 1, 1.15].map((v) => (
                <button
                  key={v}
                  className={`btn ${Math.abs(fontScale - v) < 0.01 ? "" : "secondary"}`}
                  style={{ fontSize: 11, padding: "4px 10px", flex: 1 }}
                  onClick={() => setFontScale(v)}
                >
                  {v === 0.85 ? "小" : v === 1 ? "中" : "大"}
                </button>
              ))}
            </div>
          </div>

          <div className="settings-row">
            <label className="head">参数预设</label>
            <PresetsRow />
          </div>

          <div className="settings-row" style={{ flexDirection: "row", gap: 8 }}>
            <button className="btn secondary" onClick={() => setAboutOpen(true)} style={{ flex: 1 }}>
              关于 Image Studio
            </button>
          </div>

          <div className="settings-row">
            <small style={{ color: "var(--text-dim)", fontSize: 10, lineHeight: 1.5 }}>
              快捷键:1/2/3 切工具 · 空格临时拖动 · F 重置视图 · [ ] 调笔刷 · Ctrl+Z 撤销 · Esc 取消/退出对比 · Shift+点击历史 进入对比
            </small>
          </div>
        </div>
      )}

      <Modal open={aboutOpen} onClose={() => setAboutOpen(false)} title="关于 Image Studio" width={460}>
        <div style={{ textAlign: "center", marginBottom: 18 }}>
          <div style={{
            width: 56, height: 56, margin: "0 auto 8px",
            background: "var(--gradient-primary)",
            borderRadius: 14,
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 28,
          }}>🎨</div>
          <div style={{ fontSize: 18, fontWeight: 700 }}>Image Studio</div>
          <div style={{ fontSize: 11, color: "var(--text-muted)", marginTop: 2 }}>
            v0.1.0 ·
            <span
              style={{ color: "var(--accent)", cursor: "pointer", marginLeft: 4 }}
              onClick={() => OpenExternalURL(MIT_URL).catch(() => undefined)}
            >MIT License</span>
          </div>
        </div>
        <p style={{ marginTop: 0, lineHeight: 1.6 }}>
          一个开源的 GPTCODEX 图片生成 / 编辑桌面客户端,基于 Wails(Go + React/TS)。
          所有数据(API Key、历史记录、生成图)都保存在本地机器,不上传任何服务器。
        </p>
        <div style={{ marginTop: 14, fontSize: 12, color: "var(--text-muted)", lineHeight: 1.8 }}>
          <div><strong style={{ color: "var(--text)" }}>技术栈:</strong></div>
          <div>· 后端:Go ≥ 1.25 / net/http SSE / pkg/client</div>
          <div>· 前端:React 18 + TypeScript / zustand / react-konva</div>
          <div>· 打包:Wails v2 / WebView2</div>
          <div style={{ marginTop: 6 }}><strong style={{ color: "var(--text)" }}>支持的上游:</strong></div>
          <div>· 默认 https://gptcodex.top(可在设置里切换)</div>
          <div>· 兼容 OpenAI Responses API 形态的中转站</div>
        </div>
        <div style={{ marginTop: 16, display: "flex", gap: 8 }}>
          <button
            className="btn"
            style={{ flex: 1, padding: "8px 12px", fontSize: 12 }}
            onClick={() => OpenExternalURL(REPO_URL).catch(() => undefined)}
          >
            ⌭ GitHub 仓库
          </button>
          <button
            className="btn secondary"
            style={{ flex: 1, padding: "8px 12px", fontSize: 12 }}
            onClick={() => OpenExternalURL(REPO_URL + "/issues").catch(() => undefined)}
          >
            💬 反馈 / Issues
          </button>
        </div>
        <hr style={{ borderColor: "var(--border)", margin: "16px 0 12px" }} />
        <div style={{ fontSize: 10, color: "var(--text-dim)", textAlign: "center", lineHeight: 1.6 }}>
          100% 本地数据 · 无遥测 · 无云端账户 · 无内购<br />
          Copyright © 2026 · Released under MIT
        </div>
      </Modal>
    </section>
  );
}
