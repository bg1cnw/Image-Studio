import { ImagePlus, Trash2, X } from "lucide-react";
import type {
  Mode,
  QualityValue,
  RequestPolicy,
  SizeValue,
} from "../../types/domain";
import { QUALITY_TIERS, STYLE_CHIPS } from "./panelOptions";
import { Section, Seg, SegItem } from "./panelChrome";
import {
  ASPECT_PRESETS,
  RESOLUTION_PRESETS,
  type AspectPreset,
  type ResolutionPreset,
  sizeCapabilityHint,
} from "./sizeCapabilities";

export function DesktopComposeSections({
  activeAspect,
  activeResolution,
  apiMode,
  batchCount,
  clearSources,
  currentImageSavedPath,
  handleAspectSelect,
  handleResolutionSelect,
  imageModelID,
  onRemoveSource,
  mode,
  quality,
  requestPolicy,
  selectSourceImage,
  setField,
  size,
  sources,
  styleTag,
  usesFluentUI,
  availableResolutions,
}: {
  activeAspect: AspectPreset;
  activeResolution: ResolutionPreset;
  apiMode: "responses" | "images";
  batchCount: number;
  clearSources: () => void;
  currentImageSavedPath?: string | null;
  handleAspectSelect: (aspect: AspectPreset) => void;
  handleResolutionSelect: (resolution: ResolutionPreset) => void;
  imageModelID: string;
  usesFluentUI: boolean;
  mode: Mode;
  onRemoveSource: (index: number) => void;
  quality: QualityValue;
  requestPolicy: RequestPolicy;
  selectSourceImage: () => void;
  setField: (key: "styleTag" | "quality" | "batchCount" | "size", value: any) => void;
  size: SizeValue;
  sources: Array<{ path: string; name: string }>;
  styleTag: string;
  availableResolutions: ResolutionPreset[];
}) {
  return (
    <>
      <section className="platform-card px-4 py-3.5">
        <div className="mb-1.5 flex items-center justify-between">
          <label className="text-[11px] uppercase tracking-[0.12em] text-zinc-400 dark:text-zinc-500">风格</label>
          {styleTag ? (
            <button onClick={() => setField("styleTag", "")} className="text-[11px] text-[var(--accent)] hover:opacity-80">清除</button>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-1.5">
          {STYLE_CHIPS.map((style) => {
            const active = styleTag === style.id;
            return (
              <button
                key={style.id}
                onClick={() => setField("styleTag", active ? "" : style.id)}
                className={`platform-chip px-2.5 py-1.5 text-xs ring-1 transition-colors ${
                  active
                    ? "active bg-[var(--accent-soft)] text-[var(--accent)] ring-[color:var(--accent)]/20"
                    : "text-zinc-600 dark:text-zinc-400 ring-black/[0.08] dark:ring-white/[0.08] hover:text-zinc-900 dark:hover:text-zinc-200 hover:ring-[color:var(--accent)]/30"
                } ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}
              >
                {style.label}
              </button>
            );
          })}
        </div>
      </section>

      <Section label="比例">
        <div className="grid grid-cols-3 gap-2.5">
          {ASPECT_PRESETS.map((aspect) => {
            const active = activeAspect === aspect.value;
            return (
              <button
                key={aspect.value}
                onClick={() => handleAspectSelect(aspect.value)}
                title={aspect.auto ? "让上游决定尺寸 / 比例" : aspect.value}
                className={`flex min-h-[56px] flex-col items-center justify-center gap-1 ring-1 transition-colors ${
                  active
                    ? "bg-[var(--accent-soft)] ring-[color:var(--accent)]/35"
                    : "ring-black/[0.08] dark:ring-white/[0.08] hover:ring-[color:var(--accent)]/30"
                } py-2 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}
              >
                <span
                  className={`block rounded-sm border-2 ${aspect.auto ? "border-dashed" : ""} ${
                    active ? "border-[var(--accent)]" : "border-zinc-400 dark:border-zinc-600"
                  }`}
                  style={{ width: aspect.w, height: aspect.h }}
                />
                <span className={`text-[9px] ${active ? "text-[var(--accent)]" : "text-zinc-500"}`}>{aspect.label}</span>
              </button>
            );
          })}
        </div>
      </Section>

      <Section label="分辨率">
        <Seg>
          {RESOLUTION_PRESETS.filter((item) => availableResolutions.includes(item.value)).map((item) => (
            <SegItem
              key={item.value}
              active={activeResolution === item.value}
              onClick={() => handleResolutionSelect(item.value)}
            >
              {item.label}
            </SegItem>
          ))}
        </Seg>
        {sizeCapabilityHint({ apiMode, requestPolicy, imageModelID }) ? (
          <p className="mt-1.5 text-[10px] leading-relaxed text-zinc-500 dark:text-zinc-400">
            {sizeCapabilityHint({ apiMode, requestPolicy, imageModelID })}
          </p>
        ) : null}
      </Section>

      <Section label="质量">
        <Seg>
          {QUALITY_TIERS.map((item) => (
            <SegItem
              key={item.value}
              active={quality === item.value}
              onClick={() => setField("quality", item.value as QualityValue)}
            >
              {item.label}
            </SegItem>
          ))}
        </Seg>
      </Section>

      <Section
        label="出图张数"
        trailing={<span className="font-mono-token text-[10px] text-zinc-400">{batchCount}x</span>}
      >
        <div className="grid grid-cols-3 gap-2">
          {[1, 2, 4, 6, 8, 9].map((count) => (
            <button
              key={count}
              type="button"
              onClick={() => setField("batchCount", count)}
              title={`同一提示词发起 ${count} 次请求`}
              className={`flex items-center justify-center border text-xs font-medium transition-colors ${
                batchCount === count
                  ? "border-[color:var(--accent)]/35 bg-[var(--accent-soft)] text-[var(--accent)]"
                  : "border-black/[0.08] text-zinc-600 hover:border-[color:var(--accent)]/30 hover:text-zinc-900 dark:border-white/[0.08] dark:text-zinc-400 dark:hover:text-zinc-200"
              } ${usesFluentUI ? "h-9 rounded-[8px]" : "h-9 rounded-[12px]"}`}
            >
              {count}
            </button>
          ))}
        </div>
        <p className="mt-1.5 text-[10px] leading-relaxed text-zinc-500">
          多张会并行请求,完成后在画板按网格挑图;受上游并发限制约束。
        </p>
      </Section>

      {mode === "edit" ? (
        <Section label={`源图片 / 参考图${sources.length > 0 ? ` · ${sources.length} 张` : ""}`}>
          <div className="flex flex-col gap-1.5">
            {sources.length === 0 && currentImageSavedPath ? (
              <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-xs italic text-zinc-500 dark:border-white/[0.04] dark:text-zinc-500 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}>
                (画板当前图 · 隐式源图)
              </div>
            ) : null}
            {sources.map((source, index) => (
              <div key={source.path} className={`flex items-center gap-1 border border-black/[0.06] bg-[var(--surface)] px-2.5 py-2 dark:border-white/[0.06] ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}>
                <span className="flex-1 truncate text-xs text-zinc-700 dark:text-zinc-300" title={source.path}>
                  {index + 1}. {source.name}
                </span>
                <button
                  type="button"
                  onClick={() => onRemoveSource(index)}
                  title="移除"
                  className={`-m-1 p-1 text-zinc-400 hover:bg-red-500/10 hover:text-red-400 ${usesFluentUI ? "rounded-[6px]" : "rounded-full"}`}
                >
                  <X className="w-3 h-3" />
                </button>
              </div>
            ))}
            <div className="flex gap-1.5">
              <button onClick={selectSourceImage} className={`platform-action-btn flex-1 inline-flex items-center justify-center gap-1 border border-black/[0.08] px-3 py-2 text-xs text-zinc-700 transition-colors hover:border-[color:var(--accent)]/35 hover:text-[var(--accent)] dark:border-white/[0.08] dark:text-zinc-300 ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}>
                <ImagePlus className="w-3.5 h-3.5" /> 添加图片
              </button>
              {sources.length > 0 ? (
                <button onClick={clearSources} className={`platform-action-btn inline-flex items-center gap-1 border border-black/[0.08] px-3 py-2 text-xs text-zinc-500 transition-colors hover:border-red-400/40 hover:text-red-400 dark:border-white/[0.08] ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}>
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              ) : null}
            </div>
          </div>
        </Section>
      ) : null}
    </>
  );
}
