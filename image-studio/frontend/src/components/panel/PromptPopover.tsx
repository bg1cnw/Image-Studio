import { useState } from "react";
import { useStudioStore } from "../../state/studioStore";

const PROMPT_TEMPLATES: { label: string; text: string }[] = [
  { label: "写实摄影", text: "photorealistic, professional photography, 35mm, natural lighting, sharp focus, high detail" },
  { label: "电影感", text: "cinematic, dramatic lighting, shallow depth of field, film grain, anamorphic, 2.39:1" },
  { label: "二次元", text: "anime style, vibrant colors, cel shading, detailed illustration" },
  { label: "油画", text: "oil painting, thick brush strokes, classical art style, warm tones" },
  { label: "水彩", text: "watercolor painting, soft edges, pastel colors, paper texture" },
  { label: "扁平插画", text: "flat illustration, minimalist, geometric shapes, vector style" },
  { label: "3D 渲染", text: "3D render, octane render, ray tracing, glossy, studio lighting" },
  { label: "像素风", text: "pixel art, 16-bit, retro game style, limited palette" },
];

// Popover for inserting a prompt template or recent prompt into the prompt
// textarea. Triggered from the button next to the prompt field.
export function PromptPopover({ onClose, onPick }: { onClose: () => void; onPick: (text: string) => void }) {
  const history = useStudioStore((s) => s.promptHistory);
  const [tab, setTab] = useState<"templates" | "history">("templates");

  return (
    <div className="prompt-popover" onClick={(e) => e.stopPropagation()}>
      <div className="prompt-popover-tabs">
        <button
          className={`prompt-tab ${tab === "templates" ? "active" : ""}`}
          onClick={() => setTab("templates")}
        >
          模板
        </button>
        <button
          className={`prompt-tab ${tab === "history" ? "active" : ""}`}
          onClick={() => setTab("history")}
        >
          历史 ({history.length})
        </button>
        <button className="prompt-tab close" onClick={onClose} title="关闭">×</button>
      </div>
      <div className="prompt-popover-body">
        {tab === "templates" && PROMPT_TEMPLATES.map((t) => (
          <button
            key={t.label}
            className="prompt-item"
            onClick={() => { onPick(t.text); onClose(); }}
          >
            <div className="prompt-item-title">{t.label}</div>
            <div className="prompt-item-sub">{t.text}</div>
          </button>
        ))}
        {tab === "history" && (
          history.length === 0 ? (
            <div style={{ color: "var(--text-dim)", fontSize: 11, padding: "12px 8px", textAlign: "center" }}>
              还没有提交过 prompt
            </div>
          ) : (
            history.map((p, i) => (
              <button
                key={i}
                className="prompt-item"
                onClick={() => { onPick(p); onClose(); }}
                title="点击使用"
              >
                <div className="prompt-item-sub">{p}</div>
              </button>
            ))
          )
        )}
      </div>
    </div>
  );
}
