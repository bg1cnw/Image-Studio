import { FolderOpen, RefreshCw } from "lucide-react";
import type { BatchProcessConfig, EditSourceMode } from "../../types/domain";
import { Section, Seg, SegItem } from "./panelChrome";

function sampleNames(config: BatchProcessConfig): string {
  const names = config.discoveredSources.slice(0, 3).map((item) => item.name);
  if (names.length === 0) return "未发现图片";
  if (config.discoveredSources.length <= 3) return names.join("、");
  return `${names.join("、")} 等 ${config.discoveredSources.length} 张`;
}

export function BatchProcessSection({
  currentImageSavedPath,
  editSourceMode,
  batchProcess,
  setEditSourceMode,
  setBatchProcess,
  onChooseInputDir,
  onChooseOutputDir,
  onRefreshInputDir,
  usesFluentUI = false,
}: {
  currentImageSavedPath?: string | null;
  editSourceMode: EditSourceMode;
  batchProcess: BatchProcessConfig;
  setEditSourceMode: (mode: EditSourceMode) => void;
  setBatchProcess: (next: BatchProcessConfig) => void;
  onChooseInputDir: () => void;
  onChooseOutputDir?: () => void;
  onRefreshInputDir: () => void;
  usesFluentUI?: boolean;
}) {
  const batchMode = editSourceMode === "batch";

  return (
    <Section label="源图片 / 参考图">
      <div className="space-y-3">
        <Seg>
          <SegItem
            active={!batchMode}
            onClick={() => {
              setEditSourceMode("manual");
              setBatchProcess({ ...batchProcess, enabled: false });
            }}
          >
            普通图生图
          </SegItem>
          <SegItem
            active={batchMode}
            onClick={() => {
              setEditSourceMode("batch");
              setBatchProcess({ ...batchProcess, enabled: true });
            }}
          >
            批处理
          </SegItem>
        </Seg>

        {!batchMode ? (
          <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-xs text-zinc-500 dark:border-white/[0.04] dark:text-zinc-400 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}>
            {currentImageSavedPath
              ? "当前画板图会作为隐式源图参与本次编辑，也可以继续手动添加参考图。"
              : "手动添加参考图，或从历史里挑一张结果继续编辑。"}
          </div>
        ) : (
          <div className="space-y-3">
            <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-[11px] text-zinc-500 dark:border-white/[0.04] dark:text-zinc-400 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}>
              选择一个图片目录后，会按当前 prompt 和参数逐张提交图生图任务。默认保存回原图目录，也可以单独指定输出路径。
            </div>

            <label className="space-y-1.5">
              <span className="block text-[12px] font-medium text-zinc-700 dark:text-zinc-200">输入文件夹</span>
              <input
                value={batchProcess.inputDir}
                onChange={(event) => setBatchProcess({ ...batchProcess, inputDir: event.target.value })}
                placeholder="请选择批处理输入目录"
                className={`focus-ring w-full border border-black/[0.08] bg-[var(--surface)] px-3 py-2 text-[13px] text-zinc-900 dark:border-white/[0.08] dark:text-zinc-100 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}
              />
            </label>

            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={onChooseInputDir}
                className={`inline-flex min-h-[36px] items-center gap-1.5 border border-black/[0.08] px-3 text-[12px] font-medium text-zinc-700 transition-colors hover:border-[color:var(--accent)]/35 hover:text-[var(--accent)] dark:border-white/[0.08] dark:text-zinc-300 ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}
              >
                <FolderOpen className="h-3.5 w-3.5" /> 选择目录
              </button>
              <button
                type="button"
                onClick={onRefreshInputDir}
                disabled={!batchProcess.inputDir.trim()}
                className={`inline-flex min-h-[36px] items-center gap-1.5 border border-black/[0.08] px-3 text-[12px] font-medium text-zinc-700 transition-colors hover:border-[color:var(--accent)]/35 hover:text-[var(--accent)] disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:text-zinc-300 ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}
              >
                <RefreshCw className="h-3.5 w-3.5" /> 刷新扫描
              </button>
            </div>

            <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-[11px] text-zinc-500 dark:border-white/[0.04] dark:text-zinc-400 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}>
              已扫描 {batchProcess.discoveredSources.length} 张
              <div className="mt-1 truncate">{sampleNames(batchProcess)}</div>
            </div>

            <div className="space-y-1.5">
              <span className="block text-[12px] font-medium text-zinc-700 dark:text-zinc-200">输出位置</span>
              <Seg>
                <SegItem
                  active={batchProcess.outputMode === "source_dir"}
                  onClick={() => setBatchProcess({ ...batchProcess, outputMode: "source_dir" })}
                >
                  默认保存回原目录
                </SegItem>
                <SegItem
                  active={batchProcess.outputMode === "custom_dir"}
                  onClick={() => setBatchProcess({ ...batchProcess, outputMode: "custom_dir" })}
                >
                  独立输出路径
                </SegItem>
              </Seg>
            </div>

            {batchProcess.outputMode === "custom_dir" ? (
              <div className="space-y-1.5">
                <span className="block text-[12px] font-medium text-zinc-700 dark:text-zinc-200">独立输出路径</span>
                <input
                  value={batchProcess.outputDir}
                  onChange={(event) => setBatchProcess({ ...batchProcess, outputDir: event.target.value })}
                  placeholder="请选择独立输出路径"
                  className={`focus-ring w-full border border-black/[0.08] bg-[var(--surface)] px-3 py-2 text-[13px] text-zinc-900 dark:border-white/[0.08] dark:text-zinc-100 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}
                />
                {onChooseOutputDir ? (
                  <button
                    type="button"
                    onClick={onChooseOutputDir}
                    className={`inline-flex min-h-[36px] items-center gap-1.5 border border-black/[0.08] px-3 text-[12px] font-medium text-zinc-700 transition-colors hover:border-[color:var(--accent)]/35 hover:text-[var(--accent)] dark:border-white/[0.08] dark:text-zinc-300 ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}
                  >
                    <FolderOpen className="h-3.5 w-3.5" /> 选择输出目录
                  </button>
                ) : null}
              </div>
            ) : null}

            <label className="space-y-1.5">
              <span className="block text-[12px] font-medium text-zinc-700 dark:text-zinc-200">并发数</span>
              <input
                type="number"
                min={1}
                max={9}
                value={batchProcess.concurrency}
                onChange={(event) => setBatchProcess({ ...batchProcess, concurrency: Number(event.target.value) || 1 })}
                className={`focus-ring w-full border border-black/[0.08] bg-[var(--surface)] px-3 py-2 text-[13px] text-zinc-900 dark:border-white/[0.08] dark:text-zinc-100 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}
              />
            </label>

            <div className="space-y-1.5">
              <span className="block text-[12px] font-medium text-zinc-700 dark:text-zinc-200">不同源图比例处理</span>
              <Seg>
                <SegItem
                  active={batchProcess.autoAspectResolution === ""}
                  onClick={() => setBatchProcess({ ...batchProcess, autoAspectResolution: "" })}
                >
                  沿用当前比例
                </SegItem>
                <SegItem
                  active={batchProcess.autoAspectResolution !== ""}
                  onClick={() => setBatchProcess({ ...batchProcess, autoAspectResolution: batchProcess.autoAspectResolution || "1k" })}
                >
                  按源图比例自动适配
                </SegItem>
              </Seg>
              {batchProcess.autoAspectResolution !== "" ? (
                <>
                  <div className={`border border-[color:var(--accent)]/18 bg-[var(--accent-soft)]/55 px-3 py-3 dark:border-[color:var(--accent)]/20 ${usesFluentUI ? "rounded-[12px]" : "rounded-[16px]"}`}>
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <div className="text-[12px] font-semibold text-zinc-900 dark:text-zinc-100">统一分辨率档位</div>
                        <div className="mt-0.5 text-[11px] leading-5 text-zinc-500 dark:text-zinc-400">
                          每张图按自己的比例自动适配，但都会使用这里选定的分辨率档位。
                        </div>
                      </div>
                      <span className={`shrink-0 border border-[color:var(--accent)]/25 bg-white/75 px-2.5 py-1 text-[11px] font-semibold text-[var(--accent)] dark:bg-white/10 ${usesFluentUI ? "rounded-[9px]" : "rounded-full"}`}>
                        当前 {batchProcess.autoAspectResolution.toUpperCase()}
                      </span>
                    </div>
                    <div className="mt-3 grid grid-cols-5 gap-2">
                      {(["256", "512", "1k", "2k", "4k"] as const).map((value) => (
                        <button
                          key={value}
                          type="button"
                          onClick={() => setBatchProcess({ ...batchProcess, autoAspectResolution: value })}
                          className={`border px-2 py-3 text-[12px] font-semibold transition-colors ${
                            batchProcess.autoAspectResolution === value
                              ? "border-[color:var(--accent)]/35 bg-white text-[var(--accent)] shadow-sm dark:bg-zinc-900"
                              : "border-black/[0.08] bg-white/70 text-zinc-600 hover:border-[color:var(--accent)]/30 hover:text-zinc-900 dark:border-white/[0.08] dark:bg-white/[0.03] dark:text-zinc-300"
                          } ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}
                        >
                          {value.toUpperCase()}
                        </button>
                      ))}
                    </div>
                  </div>
                  <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-[11px] text-zinc-500 dark:border-white/[0.04] dark:text-zinc-400 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}>
                    开启后，批处理会按每张源图自身宽高比自动适配尺寸，同时统一使用这里选定的分辨率档位。适合同一提示词但源图比例不同的目录批处理。
                  </div>
                </>
              ) : (
                <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-[11px] text-zinc-500 dark:border-white/[0.04] dark:text-zinc-400 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}>
                  关闭时仍沿用当前控制面板的比例和尺寸语义。
                </div>
              )}
            </div>

            <div className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-2 text-[11px] text-zinc-500 dark:border-white/[0.04] dark:text-zinc-400 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}>
              结果文件名前缀固定为 <code>processed-</code>，遇到同名会自动追加 <code>-2</code>、<code>-3</code>。
            </div>
          </div>
        )}
      </div>
    </Section>
  );
}
