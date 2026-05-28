import { useState } from "react";
import { Plus, X } from "lucide-react";
import { useStudioStore } from "../../state/studioStore";
import { useBlobURL } from "../../lib/images";
import { usePlatform } from "../../platform/context";

export function SourceStrip() {
  const sources = useStudioStore((s) => s.sources);
  const removeSource = useStudioStore((s) => s.removeSource);
  const reorderSources = useStudioStore((s) => s.reorderSources);
  const mode = useStudioStore((s) => s.mode);
  const selectSourceImage = useStudioStore((s) => s.selectSourceImage);
  const { isMac, usesFluentUI, usesAppleUI } = usePlatform();

  const [dragFrom, setDragFrom] = useState<number | null>(null);
  const [overIdx, setOverIdx] = useState<number | null>(null);

  if (mode !== "edit") return null;
  if (sources.length === 0) return null;

  return (
    <div className={`source-strip border-b border-[var(--border)] bg-[var(--toolbar)] backdrop-blur-2xl ${usesAppleUI ? "liquid-glass-bar" : ""} ${isMac ? "px-3 py-2.5" : "px-3 py-2"}`}>
      <div className={`flex ${isMac ? "items-start justify-between gap-3" : "items-center gap-2"} overflow-x-auto`}>
        <div className="min-w-0 shrink-0">
          <div className="source-strip-label text-[11px] text-zinc-500 shrink-0">参考图 {sources.length} 张</div>
          {isMac && (
            <div className="mt-0.5 text-[11px] leading-5 text-zinc-500 dark:text-zinc-400">
              图生图时常驻显示，支持拖拽排序和继续追加参考图。
            </div>
          )}
        </div>
        <div className="flex items-center gap-2 overflow-x-auto">
      {sources.map((s, i) => (
        <SourceTile
          key={s.path}
          source={s}
          index={i}
          dragFrom={dragFrom}
          overIdx={overIdx}
          setDragFrom={setDragFrom}
          setOverIdx={setOverIdx}
          reorderSources={reorderSources}
          removeSource={removeSource}
        />
      ))}
      <button
        onClick={selectSourceImage}
        title="添加参考图"
        className={`source-thumb add flex h-12 w-12 shrink-0 items-center justify-center border border-dashed border-zinc-300 text-zinc-500 transition-colors hover:border-[color:var(--accent)]/35 hover:text-[var(--accent)] dark:border-zinc-700 ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}
      >
        <Plus className="w-4 h-4" />
      </button>
        </div>
      </div>
    </div>
  );
}

function SourceTile({
  source,
  index,
  dragFrom,
  overIdx,
  setDragFrom,
  setOverIdx,
  reorderSources,
  removeSource,
}: {
  source: { path: string; name: string; imageBlob?: Blob | null; imageB64?: string };
  index: number;
  dragFrom: number | null;
  overIdx: number | null;
  setDragFrom: (v: number | null) => void;
  setOverIdx: (v: number | null) => void;
  reorderSources: (from: number, to: number) => void;
  removeSource: (index: number) => void;
}) {
  const previewURL = useBlobURL(source.imageBlob ?? null, source.imageB64 ?? null);
  const { usesFluentUI } = usePlatform();
  return (
    <div
      draggable
      onDragStart={() => setDragFrom(index)}
      onDragOver={(e) => { e.preventDefault(); setOverIdx(index); }}
      onDragLeave={() => setOverIdx(null)}
      onDrop={(e) => {
        e.preventDefault();
        if (dragFrom != null && dragFrom !== index) reorderSources(dragFrom, index);
        setDragFrom(null);
        setOverIdx(null);
      }}
      onDragEnd={() => { setDragFrom(null); setOverIdx(null); }}
      title={`${index + 1}. ${source.name}\n${source.path}`}
      className={`source-thumb relative group h-12 w-12 shrink-0 cursor-grab overflow-hidden border transition-all ${
        overIdx === index
          ? "scale-105 border-[color:var(--accent)] shadow-[0_0_0_1px_var(--accent)]"
          : "border-black/[0.06] hover:border-[color:var(--accent)]/30 dark:border-white/[0.06]"
      } ${usesFluentUI ? "rounded-[10px]" : "rounded-[14px]"}`}
    >
      <span className="absolute top-0 left-0 z-10 px-1 text-[9px] bg-zinc-950/70 text-white rounded-br">
        {index + 1}
      </span>
      {previewURL ? (
        <img
          src={previewURL}
          alt={source.name}
          loading="lazy"
          decoding="async"
          className="w-full h-full object-cover"
        />
      ) : (
        <div className="w-full h-full flex items-center justify-center text-[10px] text-zinc-500 bg-zinc-100 dark:bg-zinc-800">
          {source.name.split(".").slice(-1)[0].toUpperCase()}
        </div>
      )}
      <button
        onClick={(e) => { e.stopPropagation(); removeSource(index); }}
        title="移除"
        className="absolute top-0 right-0 w-4 h-4 flex items-center justify-center bg-zinc-950/70 text-white opacity-0 group-hover:opacity-100 hover:bg-red-500 rounded-bl transition-all"
      >
        <X className="w-2.5 h-2.5" />
      </button>
    </div>
  );
}
