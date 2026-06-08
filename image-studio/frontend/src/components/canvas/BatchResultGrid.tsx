import type { HistoryItem } from "../../types/domain";
import { historyPreviewSrc, useBlobURL } from "../../lib/images";
import { DragExportHandle } from "./DragExportHandle";

export type BatchGridSlot =
  | { type: "result"; item: HistoryItem }
  | { type: "preview"; item: HistoryItem }
  | { type: "pending"; id: string };

export function BatchResultGrid({
  items,
  slots,
  currentId,
  onSelect,
  onClose,
  showClose = true,
  title,
  selectedIds,
  onToggleSelect,
  selectionMode = false,
}: {
  items: HistoryItem[];
  slots?: BatchGridSlot[];
  currentId: string | null;
  onSelect: (item: HistoryItem) => void | Promise<void>;
  onClose: () => void;
  showClose?: boolean;
  title?: string;
  selectedIds?: Set<string>;
  onToggleSelect?: (item: HistoryItem) => void;
  selectionMode?: boolean;
}) {
  const gridSlots = slots ?? items.map((item) => ({ type: "result", item }) satisfies BatchGridSlot);
  const columns = gridSlots.length <= 2 ? 2 : gridSlots.length <= 4 ? 2 : 3;
  return (
    <div className="batch-grid-overlay">
      <div className="batch-grid-head">
        <span className="batch-grid-title">{title ?? `本批结果 · ${items.length} 张`}</span>
        {showClose ? (
          <button type="button" className="batch-grid-close" onClick={onClose} title="返回当前图">
            返回当前图
          </button>
        ) : null}
      </div>
      <div
        className="batch-grid"
        style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}
      >
        {gridSlots.map((slot, index) => {
          if (slot.type === "pending") {
            return <PendingGridTile key={slot.id} index={index} />;
          }
          return (
            <BatchGridTile
              key={slot.item.id}
              item={slot.item}
              index={index}
              active={slot.type === "result" && slot.item.id === currentId}
              preview={slot.type === "preview"}
              onSelect={onSelect}
              selected={slot.type === "result" && !!selectedIds?.has(slot.item.id)}
              onToggleSelect={onToggleSelect}
              selectionMode={selectionMode}
            />
          );
        })}
      </div>
    </div>
  );
}

function BatchGridTile({
  item,
  index,
  active,
  preview,
  onSelect,
  selected,
  onToggleSelect,
  selectionMode,
}: {
  item: HistoryItem;
  index: number;
  active: boolean;
  preview: boolean;
  onSelect: (item: HistoryItem) => void | Promise<void>;
  selected: boolean;
  onToggleSelect?: (item: HistoryItem) => void;
  selectionMode: boolean;
}) {
  const previewURL = useBlobURL(item.imageBlob ?? item.previewBlob ?? null, item.imageB64 ?? null);
  const src = historyPreviewSrc(item, previewURL);
  return (
    <div
      className={`batch-grid-tile ${active ? "active" : ""} ${preview ? "previewing" : ""} ${selected ? "selected" : ""} ${selectionMode ? "selection-mode" : ""}`}
      title={item.prompt}
    >
      <button
        type="button"
        className="batch-grid-tile-button"
        onClick={() => {
          if (selectionMode && !preview) {
            onToggleSelect?.(item);
            return;
          }
          if (!preview) void onSelect(item);
        }}
        disabled={preview}
      >
        <img
          src={src}
          alt={item.prompt || `batch result ${index + 1}`}
          loading="eager"
          decoding="async"
          draggable={false}
        />
        <span className="batch-grid-index">{index + 1}</span>
        {selectionMode && !preview ? <span className="batch-grid-check">{selected ? "已选" : "未选"}</span> : null}
        {preview ? <span className="batch-grid-meta">预览中</span> : null}
        {!preview && item.elapsedSec ? <span className="batch-grid-meta">{item.elapsedSec}s</span> : null}
      </button>
      {!preview ? <DragExportHandle item={item} className="batch-grid-drag-export" /> : null}
    </div>
  );
}

function PendingGridTile({ index }: { index: number }) {
  return (
    <div className="batch-grid-tile pending" aria-label={`等待第 ${index + 1} 张预览`}>
      <span className="batch-grid-index">{index + 1}</span>
      <span className="batch-grid-pending-ring" />
      <span className="batch-grid-pending-label">等待预览</span>
    </div>
  );
}
