import { useState } from "react";
import { X } from "lucide-react";
import { useStudioStore } from "../../state/studioStore";
import { usePlatform } from "../../lib/platformContext";

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

export function PromptPopover({ onClose, onPick }: { onClose: () => void; onPick: (text: string) => void }) {
  const history = useStudioStore((s) => s.promptHistory);
  const [tab, setTab] = useState<"templates" | "history">("templates");
  const { isWindows, usesAppleUI } = usePlatform();

  return (
    <div
      onClick={(e) => e.stopPropagation()}
      className={`absolute left-0 top-[calc(100%+0.6rem)] z-[140] flex w-[min(21.5rem,calc(100vw-3rem))] max-h-[340px] flex-col overflow-hidden border border-black/[0.08] bg-white/96 shadow-[0_28px_70px_rgb(15_23_42_/_0.22)] backdrop-blur-2xl dark:border-white/[0.08] dark:bg-[rgb(24_27_34_/_0.96)] ${usesAppleUI ? "liquid-glass-panel" : ""} ${isWindows ? "rounded-[12px]" : "rounded-[20px]"}`}
    >
      <div className="flex items-center border-b border-black/[0.06] px-1.5 py-1 dark:border-white/[0.05]">
        <button
          onClick={() => setTab("templates")}
          className={`flex-1 rounded-full px-3 py-2 text-[11px] font-semibold transition-colors ${
            tab === "templates"
              ? "bg-[var(--accent-soft)] text-[var(--accent)]"
              : "text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
          }`}
        >
          模板
        </button>
        <button
          onClick={() => setTab("history")}
          className={`flex-1 rounded-full px-3 py-2 text-[11px] font-semibold transition-colors ${
            tab === "history"
              ? "bg-[var(--accent-soft)] text-[var(--accent)]"
              : "text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
          }`}
        >
          历史 ({history.length})
        </button>
        <button
          onClick={onClose}
          title="关闭"
          className={`px-2 py-2 text-zinc-500 transition-colors hover:bg-black/[0.05] hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-white/[0.06] dark:hover:text-zinc-100 ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
        >
          <X className="w-3.5 h-3.5" />
        </button>
      </div>
      <div className="flex-1 overflow-y-auto p-2">
        {tab === "templates" && PROMPT_TEMPLATES.map((t) => (
          <button
            key={t.label}
            onClick={() => { onPick(t.text); onClose(); }}
            className={`w-full px-3 py-2.5 text-left transition-colors hover:bg-[var(--accent-soft)] ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}
          >
            <div className="mb-1 text-[12px] font-semibold text-zinc-900 dark:text-zinc-100">{t.label}</div>
            <div className="text-[11px] leading-relaxed text-zinc-500 dark:text-zinc-300">{t.text}</div>
          </button>
        ))}
        {tab === "history" && (
          history.length === 0 ? (
            <div className={`border border-dashed border-black/[0.08] px-4 py-8 text-center text-[12px] text-zinc-500 dark:border-white/[0.08] dark:text-zinc-300 ${isWindows ? "rounded-[12px]" : "rounded-[16px]"}`}>
              还没有提交过 prompt
            </div>
          ) : (
            history.map((p, i) => (
              <button
                key={i}
                onClick={() => { onPick(p); onClose(); }}
                title="点击使用"
                className={`w-full px-3 py-2.5 text-left transition-colors hover:bg-[var(--accent-soft)] ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}
              >
                <div className="text-[12px] leading-relaxed text-zinc-700 dark:text-zinc-200">{p}</div>
              </button>
            ))
          )
        )}
      </div>
    </div>
  );
}
