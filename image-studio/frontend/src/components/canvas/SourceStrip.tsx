import { useState } from "react";
import { useStudioStore } from "../../state/studioStore";

// Horizontal strip showing every reference image used for the next edit
// request. Each thumb is HTML5-draggable so the user can reorder, which
// changes the order Go sends them as input_image content blocks.
export function SourceStrip() {
  const sources = useStudioStore((s) => s.sources);
  const removeSource = useStudioStore((s) => s.removeSource);
  const reorderSources = useStudioStore((s) => s.reorderSources);
  const mode = useStudioStore((s) => s.mode);
  const selectSourceImage = useStudioStore((s) => s.selectSourceImage);

  const [dragFrom, setDragFrom] = useState<number | null>(null);
  const [overIdx, setOverIdx] = useState<number | null>(null);

  if (mode !== "edit") return null;
  if (sources.length === 0) return null;

  return (
    <div className="source-strip">
      <div className="source-strip-label">参考图 {sources.length} 张:</div>
      {sources.map((s, i) => (
        <div
          key={s.path}
          className={`source-thumb ${overIdx === i ? "over" : ""}`}
          draggable
          onDragStart={() => setDragFrom(i)}
          onDragOver={(e) => { e.preventDefault(); setOverIdx(i); }}
          onDragLeave={() => setOverIdx(null)}
          onDrop={(e) => {
            e.preventDefault();
            if (dragFrom != null && dragFrom !== i) reorderSources(dragFrom, i);
            setDragFrom(null);
            setOverIdx(null);
          }}
          onDragEnd={() => { setDragFrom(null); setOverIdx(null); }}
          title={`${i + 1}. ${s.name}\n${s.path}`}
        >
          <span className="source-thumb-idx">{i + 1}</span>
          {s.imageB64 ? (
            <img src={`data:image/png;base64,${s.imageB64}`} alt={s.name} />
          ) : (
            <div className="source-thumb-placeholder">{s.name.split(".").slice(-1)[0].toUpperCase()}</div>
          )}
          <button
            className="source-thumb-del"
            onClick={(e) => { e.stopPropagation(); removeSource(i); }}
            title="移除"
          >
            ×
          </button>
        </div>
      ))}
      <button className="source-thumb add" onClick={selectSourceImage} title="添加参考图">+</button>
    </div>
  );
}
