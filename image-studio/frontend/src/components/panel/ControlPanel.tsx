import { useState } from "react";
import { useStudioStore } from "../../state/studioStore";
import { SIZE_OPTIONS, QUALITY_OPTIONS, SizeValue, QualityValue, Mode } from "../../types/domain";
import { SettingsPanel } from "./SettingsPanel";
import { PromptPopover } from "./PromptPopover";
import { FAQModal } from "./FAQModal";

const STYLE_CHIPS: { id: string; label: string }[] = [
  { id: "cyberpunk", label: "赛博朋克" },
  { id: "anime",     label: "二次元" },
  { id: "illust",    label: "插画" },
  { id: "3d",        label: "3D 渲染" },
  { id: "chinese",   label: "国风" },
];

const ASPECT_OPTIONS: { value: SizeValue; label: string; w: number; h: number }[] = [
  { value: "1024x1024", label: "1:1",  w: 18, h: 18 },
  { value: "1024x1536", label: "3:4",  w: 14, h: 18 },
  { value: "1536x1024", label: "4:3",  w: 20, h: 15 },
  { value: "1024x1536", label: "9:16", w: 11, h: 19 },
  { value: "1536x1024", label: "16:9", w: 22, h: 12 },
];

// Quality tiers re-labelled by perceived resolution class. The underlying
// gptcodex-image quality knob is `low / medium / high`; we just relabel:
//   1K = low    (fast / cheap / less detail)
//   2K = medium (balanced — the default)
//   4K = high   (slow / expensive / best detail)
const QUALITY_TIERS: { value: QualityValue; label: string }[] = [
  { value: "low",    label: "1K" },
  { value: "medium", label: "2K" },
  { value: "high",   label: "4K" },
];

export function ControlPanel() {
  const {
    apiKey, mode, prompt, negativePrompt, size, quality, seed, batchCount, styleTag,
    sources, currentImage,
    errorMessage, isRunning, lastPayload, isTestingKey,
    setAPIKey, setField,
    selectSourceImage, removeSource, clearSources,
    submit, cancel, retryLast, testAPIKey, pushToast,
  } = useStudioStore();
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [keyOpen, setKeyOpen] = useState(!apiKey);
  const [promptPopover, setPromptPopover] = useState(false);
  const [faqOpen, setFaqOpen] = useState(false);

  const promptLen = prompt.length;
  const activeAspect = ASPECT_OPTIONS.find((a) => a.value === size)?.label ?? "1:1";

  return (
    <div className="panel">
      <div className="panel-head">
        <h1>生成控制台</h1>
      </div>

      {errorMessage && (
        <div className="error-banner">
          <div style={{ display: "flex", alignItems: "flex-start", gap: 8 }}>
            <div style={{ flex: 1, whiteSpace: "pre-wrap" }}>{errorMessage}</div>
            <button
              onClick={() => setField("errorMessage", null)}
              style={{ background: "transparent", border: 0, color: "var(--error-text)", cursor: "pointer", fontSize: 16, lineHeight: 1, padding: 0 }}
              title="关闭"
            >
              ×
            </button>
          </div>
          {lastPayload && !isRunning && (
            <div style={{ marginTop: 8 }}>
              <button className="btn secondary" onClick={retryLast} style={{ fontSize: 11, padding: "4px 10px" }}>
                ↻ 重试上次请求
              </button>
            </div>
          )}
        </div>
      )}

      <section className="prompt-wrap">
        <div className="head-row">
          <label className="head">Prompt 提示词</label>
          <span className="prompt-counter">{promptLen} / 1000</span>
        </div>
        <textarea
          className="textarea"
          value={prompt}
          maxLength={1000}
          placeholder="描述你想要生成的画面内容,越详细越好..."
          onChange={(e) => setField("prompt", e.target.value)}
        />
        <div className="prompt-foot">
          <button
            className="prompt-action-btn"
            onClick={() => setPromptPopover((v) => !v)}
            title="prompt 模板与历史"
          >
            📋 模板 / 历史
          </button>
          <span style={{ fontSize: 10, color: "var(--text-dim)" }}>Ctrl + Enter 提交</span>
        </div>
        {promptPopover && (
          <PromptPopover
            onClose={() => setPromptPopover(false)}
            onPick={(text) => {
              const current = useStudioStore.getState().prompt;
              setField("prompt", current ? `${current}\n${text}` : text);
            }}
          />
        )}
      </section>

      <section>
        <div className="head-row">
          <label className="head">风格</label>
          {styleTag && (
            <span className="head-link" onClick={() => setField("styleTag", "")}>清除</span>
          )}
        </div>
        <div className="style-chips">
          {STYLE_CHIPS.map((s) => (
            <button
              key={s.id}
              className={`style-pill ${styleTag === s.id ? "active" : ""}`}
              onClick={() => setField("styleTag", styleTag === s.id ? "" : s.id)}
            >
              {s.label}
            </button>
          ))}
        </div>
      </section>

      <section>
        <label className="head">比例</label>
        <div className="aspect-grid">
          {ASPECT_OPTIONS.map((a) => (
            <button
              key={a.label}
              className={`aspect-btn ${activeAspect === a.label ? "active" : ""}`}
              onClick={() => setField("size", a.value as SizeValue)}
            >
              <span className="aspect-icon" style={{ width: a.w, height: a.h }} />
              <span className="aspect-label">{a.label}</span>
            </button>
          ))}
        </div>
      </section>

      <section>
        <label className="head">质量</label>
        <div className="seg">
          {QUALITY_TIERS.map((q) => (
            <button
              key={q.value}
              className={`seg-item ${quality === q.value ? "active" : ""}`}
              onClick={() => setField("quality", q.value as QualityValue)}
            >
              {q.label}
            </button>
          ))}
        </div>
      </section>

      <section>
        <label className="head">数量</label>
        <div className="seg">
          {[1, 2, 4, 8].map((n) => (
            <button
              key={n}
              className={`seg-item ${batchCount === n ? "active" : ""}`}
              onClick={() => setField("batchCount", n)}
            >
              {n}
            </button>
          ))}
        </div>
      </section>

      {mode === "edit" && (
        <section>
          <label className="head">
            源图片 / 参考图
            {sources.length > 0 && (
              <span style={{ marginLeft: 6, opacity: 0.6, fontWeight: "normal" }}>· {sources.length} 张</span>
            )}
          </label>
          {sources.length === 0 && currentImage?.savedPath && (
            <div className="source-pill" style={{ opacity: 0.7 }}>
              <span className="name">(画板当前图 · 隐式源图)</span>
            </div>
          )}
          {sources.map((src, i) => (
            <div key={src.path} className="source-pill">
              <span className="name" title={src.path}>
                {i + 1}. {src.name}
              </span>
              <button onClick={() => removeSource(i)} title="移除">×</button>
            </div>
          ))}
          <div className="row">
            <button className="btn secondary" onClick={selectSourceImage}>+ 添加图片</button>
            {sources.length > 0 && <button className="btn secondary" onClick={clearSources}>清空</button>}
          </div>
        </section>
      )}

      <section>
        <button
          className="settings-toggle"
          onClick={() => setAdvancedOpen((v) => !v)}
          type="button"
        >
          <span>高级参数</span>
          <span style={{ opacity: 0.5 }}>{advancedOpen ? "▾" : "▸"} 展开设置</span>
        </button>
        {advancedOpen && (
          <div style={{ display: "flex", flexDirection: "column", gap: 10, marginTop: 8 }}>
            <textarea
              className="textarea"
              style={{ minHeight: 50 }}
              value={negativePrompt}
              placeholder="负向提示词(不希望出现的元素)..."
              onChange={(e) => setField("negativePrompt", e.target.value)}
            />
            <div className="row">
              <input
                className="input"
                type="number"
                value={seed || ""}
                placeholder="seed (留空=随机)"
                min={0}
                onChange={(e) => setField("seed", Number(e.target.value) || 0)}
              />
              <button className="btn secondary" onClick={() => setField("seed", Math.floor(Math.random() * 2_000_000_000))} title="生成随机 seed">🎲</button>
              {seed > 0 && <button className="btn secondary" onClick={() => setField("seed", 0)} title="清除">×</button>}
            </div>
            <div className="row">
              {(["generate", "edit"] as Mode[]).map((m) => (
                <label key={m} className={`radio ${mode === m ? "active" : ""}`}>
                  <input type="radio" checked={mode === m} onChange={() => setField("mode", m)} />
                  {m === "generate" ? "文生图" : "图生图"}
                </label>
              ))}
            </div>
          </div>
        )}
      </section>

      <section>
        <div className="head-row">
          <button
            className="settings-toggle"
            onClick={() => setKeyOpen((v) => !v)}
            type="button"
            style={{ flex: 1 }}
          >
            <span>API Key {apiKey && <span style={{ color: "var(--success)", fontSize: 10, marginLeft: 4 }}>● 已配置</span>}</span>
            <span style={{ opacity: 0.5 }}>{keyOpen ? "▾" : "▸"}</span>
          </button>
          <button
            className="head-link"
            onClick={() => setFaqOpen(true)}
            title="关于 API Key 分组、模型选择等"
            style={{ background: "transparent", border: 0, cursor: "pointer", padding: "0 4px", fontSize: 11 }}
          >
            ❓ FAQ
          </button>
        </div>
        {keyOpen && (
          <>
            <input
              className="input"
              type="password"
              placeholder="sk-..."
              value={apiKey}
              onChange={(e) => setAPIKey(e.target.value)}
              autoComplete="off"
              spellCheck={false}
              style={{ marginTop: 6 }}
            />
            <div className="key-hint">
              💡 选拥有 <strong>gpt-5.5</strong> 模型的分组(余额/套餐),不要 image-2 分组。
              <span onClick={() => setFaqOpen(true)} style={{ color: "var(--accent)", cursor: "pointer", marginLeft: 4 }}>详见 FAQ ›</span>
            </div>
            <button
              className="btn secondary"
              style={{ marginTop: 4, fontSize: 11, padding: "6px 10px" }}
              onClick={testAPIKey}
              disabled={!apiKey.trim() || isTestingKey}
            >
              {isTestingKey ? "测试中..." : "🔌 测试连接"}
            </button>
          </>
        )}
      </section>

      <FAQModal open={faqOpen} onClose={() => setFaqOpen(false)} />

      <div className="generate-wrap">
        {isRunning ? (
          <button className="btn danger generate-btn" onClick={cancel}>取消生成</button>
        ) : (
          <button
            className="btn generate-btn"
            onClick={submit}
            disabled={!apiKey || !prompt}
          >
            {mode === "edit" ? "编辑" : "生成"} {batchCount} 张
          </button>
        )}
        {!apiKey && (
          <div className="generate-sub">
            首次使用请在下方「API Key」区填入 sk- 开头的密钥
          </div>
        )}
      </div>

      <SettingsPanel />
    </div>
  );
}
