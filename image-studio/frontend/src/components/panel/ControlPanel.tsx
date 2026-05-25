import { lazy, Suspense, useState } from "react";
import {
  Dices, FileText, ImagePlus, ListPlus, RotateCw, Settings, Sparkles, Trash2, X,
} from "lucide-react";
import { useStudioStore } from "../../state/studioStore";
import { OpenFile } from "../../lib/runtimeHost";
import { SizeValue, QualityValue, Mode, OutputFormatValue, OUTPUT_FORMAT_OPTIONS } from "../../types/domain";
import { submitShortcutLabel } from "../../lib/platform";
import { usePlatform } from "../../lib/platformContext";

const PromptPopover = lazy(() => import("./PromptPopover").then((m) => ({ default: m.PromptPopover })));

const STYLE_CHIPS: { id: string; label: string }[] = [
  { id: "cyberpunk", label: "赛博朋克" },
  { id: "anime",     label: "二次元" },
  { id: "illust",    label: "插画" },
  { id: "3d",        label: "3D 渲染" },
  { id: "chinese",   label: "国风" },
];

// auto 项不展示具体方框形状(让上游决定),用一个虚线方框作为视觉占位。
const ASPECT_OPTIONS: { value: SizeValue; label: string; w: number; h: number; auto?: boolean }[] = [
  { value: "auto",      label: "Auto", w: 18, h: 18, auto: true },
  { value: "1024x1024", label: "1:1",  w: 18, h: 18 },
  { value: "1024x1536", label: "2:3",  w: 14, h: 20 },
  { value: "1152x2048", label: "9:16", w: 12, h: 22 },
  { value: "1536x1024", label: "3:2",  w: 22, h: 14 },
  { value: "2048x1152", label: "16:9", w: 24, h: 13 },
];

const QUALITY_TIERS: { value: QualityValue; label: string }[] = [
  { value: "auto",   label: "Auto" },
  { value: "low",    label: "1K" },
  { value: "medium", label: "2K" },
  { value: "high",   label: "4K" },
];

export function ControlPanel() {
  const {
    apiKey, mode, prompt, negativePrompt, size, quality, seed, styleTag,
    outputFormat, batchCount,
    sources, currentImage,
    errorMessage, errorRawPath, isRunning, lastPayload, isTestingKey, isOptimizingPrompt,
    apiMode, baseURL, profiles,
    noPromptRevision,
    setField, clearError, pushToast,
    selectSourceImage, removeSource, clearSources,
    openUpstreamConfig,
    submit, cancel, retryLast, optimizePrompt,
  } = useStudioStore();
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [promptPopover, setPromptPopover] = useState(false);
  const [mobileStyleOpen, setMobileStyleOpen] = useState(false);
  const [macComposeOpen, setMacComposeOpen] = useState(false);
  const { isAndroid, isAndroidPhone, isMac, isWindows, usesAndroidUI, usesAppleUI } = usePlatform();

  const promptLen = prompt.length;
  // 优化按钮只要有任一可用的 Responses profile 或当前 active 已配置就启用。
  // (实际 prompt 优化在 store.optimizePrompt 里会找到 Responses 那条 profile 跑;
  // 这里只判断 UI 是否能点。)
  const hasUsableResponsesProfile = profiles.some(
    (p) => p.apiMode === "responses" && p.baseURL.trim(),
  );
  const optimizeReady = !!(
    prompt.trim() && (hasUsableResponsesProfile || (apiKey.trim() && baseURL.trim()))
  );
  const needsUpstreamSetup = !apiKey.trim() || !baseURL.trim();
  const compactPhoneSetup = isAndroidPhone && needsUpstreamSetup;
  const compactPhoneConfigured = isAndroidPhone && !needsUpstreamSetup;
  const compactMacCompose = isMac;

  return (
    <div className={`control-panel flex ${isAndroidPhone ? "w-full shrink grow flex-col" : "w-[336px] shrink-0 flex-col"} gap-4 overflow-y-auto border-r border-[var(--border)] bg-[var(--sidebar)] px-4 py-4 backdrop-blur-2xl ${usesAppleUI ? "liquid-sidebar" : ""} ${usesAndroidUI && !isAndroidPhone ? "android-surface-pane" : ""} ${isWindows ? "pt-3" : ""}`}>
      {!compactPhoneSetup && !compactPhoneConfigured ? (
        <section className="platform-card px-4 py-3">
          <div className="flex items-start justify-between gap-3">
            <div>
              <h2
                className={`text-zinc-900 dark:text-zinc-100 ${isAndroidPhone ? "text-[18px] font-semibold tracking-[0]" : isWindows ? "text-[18px] font-semibold tracking-[0]" : "text-[20px] font-semibold tracking-[-0.02em]"}`}
                style={{ fontFamily: "var(--title-font)" }}
              >
                图像工作台
              </h2>
              {!isAndroid && (
                <p className="mt-0.5 text-[11px] leading-relaxed text-zinc-500 dark:text-zinc-400">
                  保持界面简洁，把注意力留给 prompt、参考图和结果。
                </p>
              )}
            </div>
            {!isAndroidPhone && !isMac && (
              <div className={`platform-pill bg-[var(--accent-soft)] px-2.5 py-1 text-[11px] font-medium text-[var(--accent)] ${isWindows ? "rounded-[8px]" : "rounded-2xl"}`}>
                {mode === "edit" ? "图生图" : "文生图"}
              </div>
            )}
          </div>
        </section>
      ) : null}

      {errorMessage && (
        <div className={`border border-red-500/20 bg-red-500/10 p-3 text-xs text-red-700 shadow-[var(--shadow-card)] dark:text-red-200 ${isWindows ? "rounded-[12px]" : "rounded-[18px]"}`}>
          <div className="flex items-start gap-2">
            <div className="flex-1 whitespace-pre-wrap leading-relaxed">{errorMessage}</div>
            <button
              onClick={clearError}
              className={`-m-1 p-1 text-red-400 hover:bg-red-500/10 hover:text-red-300 ${isWindows ? "rounded-[6px]" : "rounded-full"}`}
              title="关闭"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
          {(lastPayload && !isRunning) || errorRawPath ? (
            <div className="mt-2 flex flex-wrap items-center gap-1.5">
              {lastPayload && !isRunning && (
                <button
                  onClick={retryLast}
                  className={`platform-pill inline-flex items-center gap-1 bg-red-500/15 px-2.5 py-1 text-[11px] transition-colors hover:bg-red-500/25 ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                >
                  <RotateCw className="w-3 h-3" /> 重试上次请求
                </button>
              )}
              {errorRawPath && (
                <button
                  onClick={() =>
                    OpenFile(errorRawPath).catch((e: any) =>
                      pushToast(`无法打开日志:${e?.message ?? e}`, "error")
                    )
                  }
                  title={errorRawPath}
                  className={`platform-pill inline-flex items-center gap-1 px-2.5 py-1 text-[11px] ring-1 ring-red-500/30 transition-colors hover:bg-red-500/10 ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                >
                  <FileText className="w-3 h-3" /> 查看日志
                </button>
              )}
            </div>
          ) : null}
        </div>
      )}

      {/* 模式 */}
      {!compactPhoneConfigured && !compactPhoneSetup && <Section label="模式">
        <Seg>
          {(["generate", "edit"] as Mode[]).map((m) => (
            <SegItem
              key={m}
              active={mode === m}
              onClick={() => setField("mode", m)}
            >
              {m === "generate" ? "📝 文生图" : "🖼 图生图"}
            </SegItem>
          ))}
        </Seg>
      </Section>}

      {compactPhoneSetup && (
        <section className={`platform-card border border-[color:var(--accent)]/18 bg-[var(--accent-soft)] ${isAndroidPhone ? "p-3.5" : "p-4"} shadow-[var(--shadow-card)] ${isWindows ? "rounded-[12px]" : "rounded-[18px]"}`}>
          <div className="flex items-start gap-3">
            <div className={`flex h-9 w-9 shrink-0 items-center justify-center border border-[color:var(--accent)]/18 bg-white/70 dark:bg-white/[0.06] ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}>
              <Settings className="h-3.5 w-3.5 text-[var(--accent)]" />
            </div>
            <div className="min-w-0">
              <h3 className="text-[13px] font-semibold text-zinc-900 dark:text-zinc-100">先连上可用上游</h3>
              <p className="mt-0.5 text-[11px] leading-5 text-zinc-600 dark:text-zinc-300">
                先保存中转站地址和 API Key，再测试连接。连通后，这里会自动展开完整参数页。
              </p>
              <div className="mt-2.5 flex gap-2">
                <button
                  type="button"
                  onClick={openUpstreamConfig}
                  className={`liquid-primary-button inline-flex items-center gap-1.5 bg-[var(--accent)] px-3.5 py-2 text-[11px] font-medium text-white transition-colors hover:bg-[var(--accent-2)] ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                >
                  <Settings className="h-3 w-3" /> 配置上游
                </button>
                <button
                  type="button"
                  onClick={openUpstreamConfig}
                  className={`platform-action-btn inline-flex items-center gap-1.5 border border-[color:var(--accent)]/20 bg-white/70 px-3 py-2 text-[11px] text-[var(--accent)] dark:bg-white/[0.05] ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                >
                  <Sparkles className="h-3 w-3" /> 新建配置
                </button>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Prompt */}
      <section className={`platform-card relative overflow-visible ${promptPopover ? "z-30" : "z-0"} ${isAndroidPhone ? "p-3" : "p-4"}`}>
        <div className="flex items-center justify-between mb-1">
          <label className="text-[10px] uppercase tracking-[0.12em] text-zinc-400 dark:text-zinc-500">
            {mode === "edit" ? "修改要求" : compactPhoneConfigured ? "提示词" : "提示词"}
          </label>
          <span className="font-mono-token text-zinc-400 dark:text-zinc-600 tabular-nums">{promptLen}</span>
        </div>
        {compactPhoneConfigured && (
          <div className="mb-2">
            <Seg>
              {(["generate", "edit"] as Mode[]).map((m) => (
                <SegItem
                  key={m}
                  active={mode === m}
                  onClick={() => setField("mode", m)}
                >
                  {m === "generate" ? "文生图" : "图生图"}
                </SegItem>
              ))}
            </Seg>
          </div>
        )}
        <textarea
          value={prompt}
          placeholder={mode === "edit"
            ? "描述如何修改源图(例如:把背景换成夜空,人物保持不变)..."
            : "描述你想要生成的画面内容,越详细越好..."}
          onChange={(e) => setField("prompt", e.target.value)}
          className={`focus-ring w-full resize-y border border-black/[0.08] bg-[var(--surface)] px-3 py-3 leading-relaxed text-zinc-900 placeholder:text-zinc-400 dark:border-white/[0.08] dark:text-zinc-100 dark:placeholder:text-zinc-500 ${compactPhoneConfigured ? "min-h-[78px] text-[13px]" : "min-h-[124px] text-[14px]"} ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}
        />
        <div className="mt-1.5 flex items-center justify-between gap-2">
          <div className="relative shrink-0">
            <button
              type="button"
              onClick={() => setPromptPopover((v) => !v)}
              title="prompt 模板与历史"
              className={`platform-pill inline-flex items-center gap-1 px-2.5 py-1 text-[10px] transition-colors ${
                promptPopover
                  ? "bg-[var(--accent-soft)] text-[var(--accent)] ring-1 ring-[color:var(--accent)]/20"
                  : "text-zinc-500 hover:bg-[var(--accent-soft)] hover:text-[var(--accent)]"
              } ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
            >
              <ListPlus className="w-3 h-3" /> {compactPhoneConfigured ? "模板" : "模板 / 历史"}
            </button>
            {promptPopover && (
              <Suspense fallback={null}>
                <PromptPopover
                  onClose={() => setPromptPopover(false)}
                  onPick={(text) => {
                    const current = useStudioStore.getState().prompt;
                    setField("prompt", current ? `${current}\n${text}` : text);
                  }}
                />
              </Suspense>
            )}
          </div>
          <button
            type="button"
            onClick={optimizePrompt}
            disabled={!optimizeReady || isOptimizingPrompt}
            className={`platform-pill inline-flex items-center gap-1 px-2.5 py-1 text-[10px] transition-colors ${
              isOptimizingPrompt
                ? "bg-[var(--accent-soft)] text-[var(--accent)]"
                : "text-zinc-500 hover:bg-[var(--accent-soft)] hover:text-[var(--accent)]"
            } disabled:cursor-not-allowed disabled:opacity-50 ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
            title="调用 Responses/llmapi 优化当前提示词"
          >
            <Sparkles className={`w-3 h-3 ${isOptimizingPrompt ? "animate-pulse" : ""}`} />
            {isOptimizingPrompt ? "优化中..." : compactPhoneConfigured ? "优化" : "LLM 优化"}
          </button>
          <label
            title={apiMode === "responses"
              ? "勾上后 Responses API 文本模型不会优化你的 prompt,逐字传给图像模型"
              : "Images API 形态本就不优化 prompt,此开关无效"}
            className={`platform-pill inline-flex items-center gap-1.5 px-2.5 py-1 text-[10px] ring-1 transition-colors ${
              noPromptRevision
                ? "bg-[var(--accent-soft)] text-[var(--accent)] ring-[color:var(--accent)]/20"
                : "text-zinc-500 dark:text-zinc-400 ring-transparent hover:ring-black/[0.08] dark:hover:ring-white/[0.06]"
            } ${apiMode !== "responses" ? "opacity-50 cursor-not-allowed" : "cursor-pointer"} ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
          >
            <input
              type="checkbox"
              checked={noPromptRevision}
              disabled={apiMode !== "responses"}
              onChange={(e) => setField("noPromptRevision", e.target.checked)}
              className="sr-only peer"
            />
            <span className={`flex h-3.5 w-3.5 items-center justify-center rounded border transition-colors ${noPromptRevision ? "border-[var(--accent)] bg-[var(--accent)]" : "border-zinc-400 dark:border-zinc-600"}`}>
              {noPromptRevision && <span className="h-1.5 w-1.5 rounded-sm bg-white" />}
            </span>
            {compactPhoneConfigured ? "逐字" : "不优化提示词"}
          </label>
          {!isAndroidPhone && <span className="text-[10px] text-zinc-400 dark:text-zinc-600">{submitShortcutLabel}</span>}
        </div>
      </section>

      {/* 风格 */}
      {compactPhoneConfigured ? (
        <section className={`platform-card border border-black/[0.05] bg-white/70 shadow-[var(--shadow-card)] dark:border-white/[0.06] dark:bg-white/[0.03] ${isWindows ? "rounded-[12px] p-4" : "rounded-[16px] p-3"}`}>
          <button
            type="button"
            onClick={() => setMobileStyleOpen((v) => !v)}
            className="flex w-full items-center justify-between text-left"
          >
            <span className="text-[11px] uppercase tracking-[0.12em] text-zinc-400 dark:text-zinc-500">风格与质量</span>
            <span className="text-[10px] text-zinc-500">{mobileStyleOpen ? "收起 ▾" : "展开 ▸"}</span>
          </button>
          {mobileStyleOpen && (
            <div className="mt-3 flex flex-col gap-3">
              <div>
                <div className="mb-1.5 text-[11px] text-zinc-500">质量</div>
                <Seg>
                  {QUALITY_TIERS.map((q) => (
                    <SegItem
                      key={q.value}
                      active={quality === q.value}
                      onClick={() => setField("quality", q.value as QualityValue)}
                    >
                      {q.label}
                    </SegItem>
                  ))}
                </Seg>
              </div>
              <div>
                <div className="mb-1.5 flex items-center justify-between">
                  <span className="text-[11px] text-zinc-500">风格</span>
                  {styleTag && (
                    <button onClick={() => setField("styleTag", "")} className="text-[11px] text-[var(--accent)] hover:opacity-80">清除</button>
                  )}
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {STYLE_CHIPS.map((s) => {
                    const active = styleTag === s.id;
                    return (
                      <button
                        key={s.id}
                        onClick={() => setField("styleTag", active ? "" : s.id)}
                        className={`platform-chip px-2.5 py-1.5 text-xs ring-1 transition-colors ${
                          active
                            ? "active bg-[var(--accent-soft)] text-[var(--accent)] ring-[color:var(--accent)]/20"
                            : "text-zinc-600 dark:text-zinc-400 ring-black/[0.08] dark:ring-white/[0.08] hover:text-zinc-900 dark:hover:text-zinc-200 hover:ring-[color:var(--accent)]/30"
                        } ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                      >
                        {s.label}
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          )}
        </section>
      ) : !compactPhoneSetup && !compactMacCompose && <Section label="风格" trailing={styleTag ? (
        <button onClick={() => setField("styleTag", "")} className="text-[11px] text-[var(--accent)] hover:opacity-80">清除</button>
      ) : null}>
        <div className="flex flex-wrap gap-1.5">
          {STYLE_CHIPS.map((s) => {
            const active = styleTag === s.id;
            return (
              <button
                key={s.id}
                onClick={() => setField("styleTag", active ? "" : s.id)}
                className={`platform-chip px-2.5 py-1.5 text-xs ring-1 transition-colors ${
                  active
                    ? "active bg-[var(--accent-soft)] text-[var(--accent)] ring-[color:var(--accent)]/20"
                    : "text-zinc-600 dark:text-zinc-400 ring-black/[0.08] dark:ring-white/[0.08] hover:text-zinc-900 dark:hover:text-zinc-200 hover:ring-[color:var(--accent)]/30"
                } ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
              >
                {s.label}
              </button>
            );
          })}
        </div>
      </Section>}

      {/* 比例 */}
      {!compactPhoneSetup && !compactMacCompose && <Section label="比例">
        <div className={`grid gap-1.5 ${compactPhoneConfigured ? "grid-cols-3" : "grid-cols-6"}`}>
          {ASPECT_OPTIONS.map((a) => {
            const active = size === a.value;
            return (
              <button
                key={a.value}
                onClick={() => setField("size", a.value)}
                title={a.auto ? "让上游决定尺寸 / 比例" : a.value}
                className={`flex flex-col items-center gap-1 ring-1 transition-colors ${
                  active
                    ? "bg-[var(--accent-soft)] ring-[color:var(--accent)]/35"
                    : "ring-black/[0.08] dark:ring-white/[0.08] hover:ring-[color:var(--accent)]/30"
                } ${compactPhoneConfigured ? "min-h-[48px] py-2" : "py-2"} ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}
              >
                <span
                  className={`block rounded-sm border-2 ${a.auto ? "border-dashed" : ""} ${
                    active ? "border-[var(--accent)]" : "border-zinc-400 dark:border-zinc-600"
                  }`}
                  style={{ width: a.w, height: a.h }}
                />
                <span className={`text-[10px] ${active ? "text-[var(--accent)]" : "text-zinc-500"}`}>{a.label}</span>
              </button>
            );
          })}
        </div>
      </Section>}

      {/* 质量 */}
      {!compactPhoneSetup && !compactPhoneConfigured && !compactMacCompose && <Section label="质量">
        <Seg>
          {QUALITY_TIERS.map((q) => (
            <SegItem
              key={q.value}
              active={quality === q.value}
              onClick={() => setField("quality", q.value as QualityValue)}
            >
              {q.label}
            </SegItem>
          ))}
        </Seg>
      </Section>}

      {!compactPhoneSetup && !compactMacCompose && <Section
        label="出图张数"
        trailing={<span className="font-mono-token text-[10px] text-zinc-400">{batchCount}x</span>}
      >
        <div className={`grid gap-1.5 ${compactPhoneConfigured ? "grid-cols-6" : "grid-cols-3"}`}>
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
              } ${compactPhoneConfigured ? "h-9 rounded-[12px]" : isWindows ? "h-9 rounded-[8px]" : "h-9 rounded-[12px]"}`}
            >
              {count}
            </button>
          ))}
        </div>
        {!compactPhoneConfigured && (
          <p className="mt-1.5 text-[10px] leading-relaxed text-zinc-500">
            多张会并行请求,完成后在画板按网格挑图;受上游并发限制约束。
          </p>
        )}
      </Section>}

      {compactMacCompose && (
        <section className={`platform-card border border-black/[0.05] bg-white/70 shadow-[var(--shadow-card)] dark:border-white/[0.06] dark:bg-white/[0.03] ${isWindows ? "rounded-[12px] p-4" : "rounded-[18px] p-4"}`}>
          <button
            type="button"
            onClick={() => setMacComposeOpen((v) => !v)}
            className="flex w-full items-center justify-between text-left"
          >
            <div>
              <div className="text-[11px] uppercase tracking-[0.12em] text-zinc-400 dark:text-zinc-500">创作参数</div>
              <div className="mt-1 text-[12px] text-zinc-500">
                {styleTag ? `风格 ${STYLE_CHIPS.find((item) => item.id === styleTag)?.label ?? styleTag}` : "默认风格"} · {size === "auto" ? "Auto 比例" : ASPECT_OPTIONS.find((item) => item.value === size)?.label ?? size} · {QUALITY_TIERS.find((item) => item.value === quality)?.label ?? quality} · {batchCount} 张
              </div>
            </div>
            <span className="text-[10px] text-zinc-500">{macComposeOpen ? "收起 ▾" : "展开 ▸"}</span>
          </button>
          {macComposeOpen && (
            <div className="mt-4 flex flex-col gap-4">
              <div>
                <div className="mb-1.5 flex items-center justify-between">
                  <span className="text-[11px] text-zinc-500">风格</span>
                  {styleTag && (
                    <button onClick={() => setField("styleTag", "")} className="text-[11px] text-[var(--accent)] hover:opacity-80">清除</button>
                  )}
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {STYLE_CHIPS.map((s) => {
                    const active = styleTag === s.id;
                    return (
                      <button
                        key={s.id}
                        onClick={() => setField("styleTag", active ? "" : s.id)}
                        className={`platform-chip px-2.5 py-1.5 text-xs ring-1 transition-colors ${
                          active
                            ? "active bg-[var(--accent-soft)] text-[var(--accent)] ring-[color:var(--accent)]/20"
                            : "text-zinc-600 dark:text-zinc-400 ring-black/[0.08] dark:ring-white/[0.08] hover:text-zinc-900 dark:hover:text-zinc-200 hover:ring-[color:var(--accent)]/30"
                        } ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                      >
                        {s.label}
                      </button>
                    );
                  })}
                </div>
              </div>
              <div>
                <div className="mb-1.5 text-[11px] text-zinc-500">比例</div>
                <div className="grid grid-cols-6 gap-1.5">
                  {ASPECT_OPTIONS.map((a) => {
                    const active = size === a.value;
                    return (
                      <button
                        key={a.value}
                        onClick={() => setField("size", a.value)}
                        title={a.auto ? "让上游决定尺寸 / 比例" : a.value}
                        className={`flex flex-col items-center gap-1 py-2 ring-1 transition-colors ${
                          active
                            ? "bg-[var(--accent-soft)] ring-[color:var(--accent)]/35"
                            : "ring-black/[0.08] dark:ring-white/[0.08] hover:ring-[color:var(--accent)]/30"
                        } ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}
                      >
                        <span
                          className={`block rounded-sm border-2 ${a.auto ? "border-dashed" : ""} ${
                            active ? "border-[var(--accent)]" : "border-zinc-400 dark:border-zinc-600"
                          }`}
                          style={{ width: a.w, height: a.h }}
                        />
                        <span className={`text-[10px] ${active ? "text-[var(--accent)]" : "text-zinc-500"}`}>{a.label}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
              <div>
                <div className="mb-1.5 text-[11px] text-zinc-500">质量</div>
                <Seg>
                  {QUALITY_TIERS.map((q) => (
                    <SegItem
                      key={q.value}
                      active={quality === q.value}
                      onClick={() => setField("quality", q.value as QualityValue)}
                    >
                      {q.label}
                    </SegItem>
                  ))}
                </Seg>
              </div>
              <div>
                <div className="mb-1.5 flex items-center justify-between">
                  <span className="text-[11px] text-zinc-500">出图张数</span>
                  <span className="font-mono-token text-[10px] text-zinc-400">{batchCount}x</span>
                </div>
                <div className="grid grid-cols-3 gap-1.5">
                  {[1, 2, 4, 6, 8, 9].map((count) => (
                    <button
                      key={count}
                      type="button"
                      onClick={() => setField("batchCount", count)}
                      title={`同一提示词发起 ${count} 次请求`}
                      className={`flex h-9 items-center justify-center border text-xs font-medium transition-colors ${
                        batchCount === count
                          ? "border-[color:var(--accent)]/35 bg-[var(--accent-soft)] text-[var(--accent)]"
                          : "border-black/[0.08] text-zinc-600 hover:border-[color:var(--accent)]/30 hover:text-zinc-900 dark:border-white/[0.08] dark:text-zinc-400 dark:hover:text-zinc-200"
                      } ${isWindows ? "rounded-[8px]" : "rounded-[12px]"}`}
                    >
                      {count}
                    </button>
                  ))}
                </div>
              </div>
              {mode === "edit" && (
                <div>
                  <div className="mb-1.5 flex items-center justify-between">
                    <span className="text-[11px] text-zinc-500">源图片 / 参考图</span>
                    <span className="text-[10px] text-zinc-400">
                      {sources.length > 0 ? `${sources.length} 张` : currentImage?.savedPath ? "已就绪" : "未添加"}
                    </span>
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-[11px] text-zinc-500 dark:border-white/[0.04] dark:text-zinc-400 ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}>
                      {sources.length > 0
                        ? "已添加显式参考图，可继续追加、替换或拖入更多图片。"
                        : currentImage?.savedPath
                          ? "当前画板图会作为隐式源图参与本次编辑。"
                          : "先添加一张参考图，或从历史里挑一张结果继续编辑。"}
                    </div>
                    <div className="flex gap-1.5">
                      <button
                        onClick={selectSourceImage}
                        className={`platform-action-btn flex-1 inline-flex items-center justify-center gap-1 border border-black/[0.08] px-3 py-2 text-xs text-zinc-700 transition-colors hover:border-[color:var(--accent)]/35 hover:text-[var(--accent)] dark:border-white/[0.08] dark:text-zinc-300 ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                      >
                        <ImagePlus className="w-3.5 h-3.5" /> 添加图片
                      </button>
                      {sources.length > 0 && (
                        <button
                          onClick={clearSources}
                          className={`platform-action-btn inline-flex items-center gap-1 border border-black/[0.08] px-3 py-2 text-xs text-zinc-500 transition-colors hover:border-red-400/40 hover:text-red-400 dark:border-white/[0.08] ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
        </section>
      )}

      {/* 源图(只在 edit 模式)*/}
      {mode === "edit" && !compactPhoneSetup && !compactMacCompose && (
        <Section
          label={`源图片 / 参考图${sources.length > 0 ? ` · ${sources.length} 张` : ""}`}
        >
          <div className="flex flex-col gap-1.5">
            {sources.length === 0 && currentImage?.savedPath && (
              <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-xs italic text-zinc-500 dark:border-white/[0.04] dark:text-zinc-500 ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}>
                (画板当前图 · 隐式源图)
              </div>
            )}
            {sources.map((src, i) => (
              <div key={src.path} className={`flex items-center gap-1 border border-black/[0.06] bg-[var(--surface)] px-2.5 py-2 dark:border-white/[0.06] ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}>
                <span className="flex-1 text-xs text-zinc-700 dark:text-zinc-300 truncate" title={src.path}>
                  {i + 1}. {src.name}
                </span>
                <button
                  onClick={() => removeSource(i)}
                  title="移除"
                  className={`-m-1 p-1 text-zinc-400 hover:bg-red-500/10 hover:text-red-400 ${isWindows ? "rounded-[6px]" : "rounded-full"}`}
                >
                  <X className="w-3 h-3" />
                </button>
              </div>
            ))}
            <div className="flex gap-1.5">
              <button onClick={selectSourceImage} className={`platform-action-btn flex-1 inline-flex items-center justify-center gap-1 border border-black/[0.08] px-3 py-2 text-xs text-zinc-700 transition-colors hover:border-[color:var(--accent)]/35 hover:text-[var(--accent)] dark:border-white/[0.08] dark:text-zinc-300 ${isWindows ? "rounded-[8px]" : "rounded-full"}`}>
                <ImagePlus className="w-3.5 h-3.5" /> 添加图片
              </button>
              {sources.length > 0 && (
                <button onClick={clearSources} className={`platform-action-btn inline-flex items-center gap-1 border border-black/[0.08] px-3 py-2 text-xs text-zinc-500 transition-colors hover:border-red-400/40 hover:text-red-400 dark:border-white/[0.08] ${isWindows ? "rounded-[8px]" : "rounded-full"}`}>
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          </div>
        </Section>
      )}

      {/* 高级参数(可折叠)*/}
      {!compactPhoneSetup && <section>
        <button
          onClick={() => setAdvancedOpen((v) => !v)}
          type="button"
          className={`platform-card flex w-full items-center justify-between border border-black/[0.05] bg-white/70 px-4 py-3 text-xs text-zinc-500 transition-colors hover:text-zinc-900 dark:border-white/[0.06] dark:bg-white/[0.03] dark:hover:text-zinc-200 ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}
        >
          <span className="uppercase tracking-[0.12em]">高级参数</span>
          <span className="text-[10px] opacity-60">{advancedOpen ? "收起 ▾" : "展开 ▸"}</span>
        </button>
        {advancedOpen && (
          <div className={`platform-card mt-2 flex flex-col gap-2 border border-black/[0.05] bg-white/70 p-4 shadow-[var(--shadow-card)] dark:border-white/[0.06] dark:bg-white/[0.03] ${isWindows ? "rounded-[12px]" : "rounded-[18px]"}`}>
            <textarea
              value={negativePrompt}
              placeholder="负向提示词(不希望出现的元素)..."
              onChange={(e) => setField("negativePrompt", e.target.value)}
              className={`focus-ring min-h-[56px] w-full resize-y border border-black/[0.08] bg-[var(--surface)] px-3 py-2 text-xs text-zinc-900 placeholder:text-zinc-400 dark:border-white/[0.08] dark:text-zinc-100 dark:placeholder:text-zinc-500 ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}
            />
            <div>
              <label className="mb-1 block text-[10px] uppercase tracking-[0.12em] text-zinc-400 dark:text-zinc-500">输出格式</label>
              <Seg>
                {OUTPUT_FORMAT_OPTIONS.map((f) => (
                  <SegItem
                    key={f.value}
                    active={outputFormat === f.value}
                    onClick={() => setField("outputFormat", f.value as OutputFormatValue)}
                  >
                    {f.label}
                  </SegItem>
                ))}
              </Seg>
              <p className="text-[10px] text-zinc-500 mt-1">JPEG/WebP 体积更小;落盘扩展名 jpeg→.jpg</p>
            </div>
            <div className="flex gap-1.5">
              <input
                type="number"
                value={seed || ""}
                placeholder="seed (留空=随机)"
                min={0}
                onChange={(e) => setField("seed", Number(e.target.value) || 0)}
                className={`focus-ring flex-1 border border-black/[0.08] bg-[var(--surface)] px-3 py-2 text-xs font-mono-token text-zinc-900 placeholder:text-zinc-400 dark:border-white/[0.08] dark:text-zinc-100 dark:placeholder:text-zinc-500 ${isWindows ? "rounded-[10px]" : "rounded-[14px]"}`}
              />
              <button
                onClick={() => setField("seed", Math.floor(Math.random() * 2_000_000_000))}
                title="随机 seed"
                className={`platform-action-btn border border-black/[0.08] px-2.5 py-2 text-zinc-600 transition-colors hover:border-[color:var(--accent)]/35 hover:text-[var(--accent)] dark:border-white/[0.08] dark:text-zinc-400 ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
              >
                <Dices className="w-3.5 h-3.5" />
              </button>
              {seed > 0 && (
                <button
                  onClick={() => setField("seed", 0)}
                  title="清除"
                  className={`platform-action-btn border border-black/[0.08] px-2.5 py-2 text-zinc-500 transition-colors hover:border-red-400/40 hover:text-red-400 dark:border-white/[0.08] ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          </div>
        )}
      </section>}

      {/* 生成按钮 */}
      <div className="sticky bottom-0 -mx-4 mt-auto bg-gradient-to-t from-[var(--sidebar)] via-[color:var(--sidebar)]/96 to-transparent px-4 pb-4 pt-2">
        {(!apiKey || !baseURL) && !compactPhoneSetup && (
          <div className={`mb-2 border border-[color:var(--accent)]/18 bg-[var(--accent-soft)] px-3 py-2.5 text-center text-[11px] leading-relaxed text-[var(--accent)] ${isWindows ? "rounded-[10px]" : "rounded-[16px]"}`}>
            <div className="font-medium">还没有可用上游配置</div>
            <div className="mt-1 opacity-90">
              先配置 BASE_URL 和 API Key，才能测试连接或开始生成。
            </div>
            <button
              type="button"
              onClick={openUpstreamConfig}
              className={`mt-2 inline-flex items-center gap-1.5 border border-[color:var(--accent)]/22 bg-white/70 px-3 py-1.5 text-[11px] font-medium text-[var(--accent)] transition-colors hover:bg-white/90 dark:bg-white/[0.06] ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
            >
              <Settings className="h-3.5 w-3.5" /> 配置上游
            </button>
          </div>
        )}
        {compactPhoneSetup ? (
          <button
            type="button"
            onClick={openUpstreamConfig}
            className={`liquid-primary-button w-full bg-[var(--accent)] py-3 font-semibold text-white transition-colors hover:bg-[var(--accent-2)] ${isWindows ? "rounded-[10px]" : "rounded-full"}`}
          >
            配置上游
          </button>
        ) : isRunning ? (
          <button
            onClick={cancel}
            className={`w-full border border-red-500/30 bg-red-500/10 py-3 font-medium text-red-500 transition-colors hover:bg-red-500/16 ${isWindows ? "rounded-[10px]" : "rounded-full"}`}
          >
            取消生成
          </button>
        ) : (
          <button
            onClick={submit}
            disabled={!apiKey || !prompt}
            className={`liquid-primary-button w-full bg-[var(--accent)] py-3 font-semibold text-white transition-colors hover:bg-[var(--accent-2)] disabled:cursor-not-allowed disabled:bg-zinc-300 disabled:text-zinc-500 dark:disabled:bg-zinc-800 ${isWindows ? "rounded-[10px]" : "rounded-full"}`}
          >
            {mode === "edit" ? "编辑" : "生成"}
          </button>
        )}
      </div>
    </div>
  );
}

// ---- 内部 helpers ----

function Section({
  label, trailing, children,
}: {
  label: React.ReactNode;
  trailing?: React.ReactNode;
  children: React.ReactNode;
}) {
  const { isAndroidPhone, isWindows } = usePlatform();
  return (
    <section className={`platform-card border border-black/[0.05] bg-white/70 shadow-[var(--shadow-card)] dark:border-white/[0.06] dark:bg-white/[0.03] ${isAndroidPhone ? "p-3" : "p-4"} ${isWindows ? "rounded-[12px]" : "rounded-[18px]"}`}>
      <div className={`flex items-center justify-between ${isAndroidPhone ? "mb-1" : "mb-1.5"}`}>
        <label className="text-[11px] uppercase tracking-[0.12em] text-zinc-400 dark:text-zinc-500">{label}</label>
        {trailing}
      </div>
      {children}
    </section>
  );
}

function Seg({ children }: { children: React.ReactNode }) {
  const { isWindows } = usePlatform();
  return (
    <div className={`platform-seg flex gap-1 bg-black/[0.04] p-0.5 ring-1 ring-black/[0.05] dark:bg-white/[0.06] dark:ring-white/[0.06] ${isWindows ? "rounded-[10px]" : "rounded-full"}`}>
      {children}
    </div>
  );
}

function SegItem({ active, onClick, children }: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  const { isWindows } = usePlatform();
  return (
    <button
      type="button"
      onClick={onClick}
        className={`platform-chip flex-1 px-2 py-1.5 text-xs font-medium transition-colors ${
        active
          ? "active bg-white text-zinc-900 shadow-sm dark:bg-zinc-900 dark:text-zinc-100"
          : "text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-200"
      } ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
    >
      {children}
    </button>
  );
}
