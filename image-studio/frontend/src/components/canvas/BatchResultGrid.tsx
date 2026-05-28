import type { HistoryItem } from "../../types/domain";
import { useBlobURL } from "../../lib/images";

export function BatchResultGrid({
  items,
  currentId,
  onSelect,
  onClose,
}: {
  items: HistoryItem[];
  currentId: string | null;
  onSelect: (item: HistoryItem) => void | Promise<void>;
  onClose: () => void;
}) {
  const columns = items.length <= 2 ? 2 : items.length <= 4 ? 2 : 3;
  return (
    <div className="batch-grid-overlay">
      <div className="batch-grid-head">
        <span className="batch-grid-title">本批结果 · {items.length} 张</span>
        <button type="button" className="batch-grid-close" onClick={onClose} title="返回当前图">
          返回当前图
        </button>
      </div>
      <div
        className="batch-grid"
        style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}
      >
        {items.map((item, index) => (
          <BatchGridTile
            key={item.id}
            item={item}
            index={index}
            active={item.id === currentId}
            onSelect={onSelect}
          />
        ))}
      </div>
    </div>
  );
}

function BatchGridTile({
  item,
  index,
  active,
  onSelect,
}: {
  item: HistoryItem;
  index: number;
  active: boolean;
  onSelect: (item: HistoryItem) => void | Promise<void>;
}) {
  const previewURL = useBlobURL(item.imageBlob ?? item.previewBlob ?? null, item.imageB64 ?? null);
  return (
    <button
      type="button"
      className={`batch-grid-tile ${active ? "active" : ""}`}
      onClick={() => void onSelect(item)}
      title={item.prompt}
    >
      <img
        src={previewURL ?? `data:image/png;base64,${item.imageB64}`}
        alt={item.prompt || `batch result ${index + 1}`}
        loading="eager"
        decoding="async"
        draggable={false}
      />
      <span className="batch-grid-index">{index + 1}</span>
      {item.elapsedSec && <span className="batch-grid-meta">{item.elapsedSec}s</span>}
    </button>
  );
}
