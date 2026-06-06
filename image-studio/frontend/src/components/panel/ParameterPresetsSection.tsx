import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronRight, FolderCog, Plus, Save } from "lucide-react";
import { findMatchingPresetId, nextDefaultPresetName, pickPresetStateSnapshot } from "../../lib/presets";
import { usePlatform } from "../../platform/context";
import { useStudioStore } from "../../state/studioStore";
import type { Preset } from "../../types/domain";
import { PresetManagerModal } from "./PresetManagerModal";
import { PresetSaveModal } from "./PresetSaveModal";
import { STYLE_CHIPS } from "./panelOptions";

const QUALITY_LABELS: Record<string, string> = {
  auto: "Auto",
  low: "低",
  medium: "中",
  high: "高",
  standard: "标准",
  hd: "HD",
};

export function ParameterPresetsSection({
  variant = "desktop",
}: {
  variant?: "desktop" | "android";
}) {
  const state = useStudioStore();
  const {
    presets,
    savePreset,
    overwritePreset,
    updatePreset,
    applyPreset,
    deletePreset,
  } = state;
  const { usesFluentUI } = usePlatform();
  const [expanded, setExpanded] = useState(true);
  const [selectedPresetId, setSelectedPresetId] = useState<string | null>(null);
  const [managerOpen, setManagerOpen] = useState(false);
  const [saveModalMode, setSaveModalMode] = useState<"save-current" | "new" | null>(null);
  const currentSnapshot = pickPresetStateSnapshot(state);
  const matchedPresetId = findMatchingPresetId(presets, currentSnapshot);
  const selectedPreset = presets.find((preset) => preset.id === selectedPresetId) ?? null;
  const currentPreset = selectedPreset ?? presets.find((preset) => preset.id === matchedPresetId) ?? null;
  const isAndroid = variant === "android";
  const cardRadius = isAndroid ? "rounded-[18px]" : usesFluentUI ? "rounded-[10px]" : "rounded-[16px]";
  const buttonRadius = usesFluentUI ? "rounded-[8px]" : "rounded-full";
  const suggestedName = useMemo(() => nextDefaultPresetName(presets), [presets]);

  useEffect(() => {
    if (selectedPresetId && presets.some((preset) => preset.id === selectedPresetId)) return;
    setSelectedPresetId(matchedPresetId);
  }, [selectedPresetId, matchedPresetId, presets]);

  useEffect(() => {
    if (presets.length === 0) setExpanded(true);
  }, [presets.length]);

  function handleApplyPreset(id: string) {
    setSelectedPresetId(id);
    applyPreset(id);
  }

  function handleSaveAsNewPreset(name: string) {
    const id = savePreset(name);
    if (id) {
      setSelectedPresetId(id);
      setExpanded(true);
    }
    return id;
  }

  function handleOverwritePreset(id: string) {
    return overwritePreset(id);
  }

  function handleUpdatePreset(id: string, patch: Partial<Omit<Preset, "id">>) {
    return updatePreset(id, patch);
  }

  const summary = buildPresetSummary({
    matchedPresetId,
    matchedPresetName: presets.find((preset) => preset.id === matchedPresetId)?.name ?? null,
    selectedPreset,
  });

  return (
    <>
      <section className={`border border-black/[0.06] bg-[var(--surface)] px-3 py-3 dark:border-white/[0.08] ${cardRadius}`}>
        <button
          type="button"
          onClick={() => setExpanded((value) => !value)}
          className="flex w-full items-start justify-between gap-3 text-left"
        >
          <span className="min-w-0">
            <span className="block text-[11px] font-semibold uppercase tracking-[0.12em] text-zinc-500 dark:text-zinc-400">
              参数预设
            </span>
            <span className="mt-1 block text-[13px] font-medium text-zinc-900 dark:text-zinc-100">
              {summary.title}
            </span>
            <span className="mt-1 block text-[11px] leading-5 text-zinc-500 dark:text-zinc-400">
              {summary.detail}
            </span>
          </span>
          <span className={`mt-1 shrink-0 text-zinc-400 dark:text-zinc-500 ${buttonRadius}`}>
            {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </span>
        </button>

        {expanded ? (
          <div className={`mt-3 flex flex-col ${isAndroid ? "gap-2.5" : "gap-2"}`}>
            {presets.length > 0 ? (
              <div className={`flex flex-col ${isAndroid ? "gap-2.5" : "gap-1.5"}`}>
                {presets.map((preset) => {
                  const matched = preset.id === matchedPresetId;
                  const selected = preset.id === selectedPresetId;
                  return (
                    <button
                      key={preset.id}
                      type="button"
                      onClick={() => handleApplyPreset(preset.id)}
                      className={`border px-3 text-left transition-colors ${
                        selected || matched
                          ? "border-[color:var(--accent)]/35 bg-[var(--accent-soft)]"
                          : "border-black/[0.08] hover:border-[color:var(--accent)]/25 dark:border-white/[0.08]"
                      } ${cardRadius} ${isAndroid ? "py-3.5" : "py-2.5"}`}
                    >
                      <span className="flex items-center gap-2">
                        <span className="min-w-0 text-[13px] font-medium text-zinc-900 dark:text-zinc-100">{preset.name}</span>
                        {selected ? <PresetBadge variant="selected">已选</PresetBadge> : null}
                        {matched ? <PresetBadge variant="matched">匹配</PresetBadge> : null}
                      </span>
                      <span className="mt-1 block text-[11px] leading-5 text-zinc-500 dark:text-zinc-400">
                        {describePreset(preset)}
                      </span>
                    </button>
                  );
                })}
              </div>
            ) : (
              <div className={`border border-dashed border-black/[0.12] px-3 py-3 text-zinc-500 dark:border-white/[0.1] dark:text-zinc-400 ${cardRadius} ${isAndroid ? "text-[12px]" : "text-[11px]"}`}>
                还没有保存任何预设。先写好提示词和参数，再新建一条。
              </div>
            )}

            <div className={`grid ${isAndroid ? "grid-cols-1 gap-2.5" : "grid-cols-3 gap-2"}`}>
              <ActionButton icon={<Save className="h-3.5 w-3.5" />} onClick={() => setSaveModalMode("save-current")}>
                保存当前预设
              </ActionButton>
              <ActionButton icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setSaveModalMode("new")}>
                新建预设
              </ActionButton>
              <ActionButton icon={<FolderCog className="h-3.5 w-3.5" />} onClick={() => setManagerOpen(true)}>
                预设管理
              </ActionButton>
            </div>
          </div>
        ) : null}
      </section>

      <PresetSaveModal
        open={saveModalMode !== null}
        mode={saveModalMode ?? "new"}
        currentPreset={currentPreset ? { id: currentPreset.id, name: currentPreset.name } : null}
        suggestedName={suggestedName}
        onClose={() => setSaveModalMode(null)}
        onOverwritePreset={handleOverwritePreset}
        onSaveAsNewPreset={handleSaveAsNewPreset}
      />

      <PresetManagerModal
        open={managerOpen}
        presets={presets}
        selectedPresetId={selectedPresetId}
        onClose={() => setManagerOpen(false)}
        onDeletePreset={(id) => {
          if (selectedPresetId === id) setSelectedPresetId(null);
          deletePreset(id);
        }}
        onUpdatePreset={handleUpdatePreset}
      />
    </>
  );
}

function ActionButton({
  children,
  icon,
  onClick,
}: {
  children: ReactNode;
  icon: ReactNode;
  onClick: () => void;
}) {
  const { usesFluentUI } = usePlatform();

  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex min-h-[38px] items-center justify-center gap-1.5 border border-black/[0.08] px-3 text-[12px] font-medium text-zinc-600 transition-colors hover:border-[color:var(--accent)]/30 hover:text-[var(--accent)] dark:border-white/[0.08] dark:text-zinc-300 ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}
    >
      {icon}
      {children}
    </button>
  );
}

function PresetBadge({
  children,
  variant,
}: {
  children: ReactNode;
  variant: "matched" | "selected";
}) {
  const { usesFluentUI } = usePlatform();
  const colorClasses = variant === "selected"
    ? "border-[color:var(--accent)]/20 bg-white/70 text-[var(--accent)] dark:bg-black/20"
    : "border-emerald-400/20 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300";

  return (
    <span className={`shrink-0 border px-1.5 py-0.5 text-[10px] font-medium ${colorClasses} ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}>
      {children}
    </span>
  );
}

function buildPresetSummary({
  matchedPresetId,
  matchedPresetName,
  selectedPreset,
}: {
  matchedPresetId: string | null;
  matchedPresetName: string | null;
  selectedPreset: Preset | null;
}) {
  if (selectedPreset && selectedPreset.id === matchedPresetId) {
    return {
      title: `当前使用「${selectedPreset.name}」`,
      detail: "当前参数与已选预设完全一致，保存时可以直接覆盖它。",
    };
  }
  if (selectedPreset) {
    return {
      title: `已选「${selectedPreset.name}」`,
      detail: "你可以继续调参数，再用“保存当前预设”覆盖这条已选预设，或另存为新的预设。",
    };
  }
  if (matchedPresetName) {
    return {
      title: `当前匹配「${matchedPresetName}」`,
      detail: "当前参数正好匹配一条已有预设，展开后可以直接继续切换。",
    };
  }
  return {
    title: "还没有选中预设",
    detail: "展开后可以直接切换多个预设，或把当前参数保存成新的预设。",
  };
}

function describePreset(preset: Preset): string {
  const styleLabel = preset.styleTag
    ? STYLE_CHIPS.find((item) => item.id === preset.styleTag)?.label ?? preset.styleTag
    : "默认风格";
  return [
    preset.size === "auto" ? "Auto 尺寸" : preset.size,
    `质量 ${QUALITY_LABELS[preset.quality] ?? preset.quality}`,
    preset.outputFormat ? preset.outputFormat.toUpperCase() : "PNG",
    `风格 ${styleLabel}`,
    `${preset.batchCount} 张`,
  ].join(" · ");
}
