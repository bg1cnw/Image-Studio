import type { HistoryItem } from "../../types/domain";
import { buildHistoryItemDragExport, writeImageFileDragData } from "../../lib/dragExport.ts";

export function DragExportHandle({
  item,
  className = "",
  sourceURL,
}: {
  item: Pick<HistoryItem, "id" | "mode" | "outputFormat" | "savedPath" | "imageId" | "fullUrl" | "imageB64" | "previewOnly">;
  className?: string;
  sourceURL?: string | null;
}) {
  const spec = buildHistoryItemDragExport(item, sourceURL);
  if (!spec) return null;

  const classes = ["image-drag-export", className].filter(Boolean).join(" ");
  const label = "拖到文件夹复制";

  return (
    <a
      href={spec.href}
      download={spec.fileName}
      draggable
      className={classes}
      title={`${label} · ${spec.fileName}`}
      aria-label={`${label} · ${spec.fileName}`}
      onMouseDown={(event) => event.stopPropagation()}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
      }}
      onDragStart={(event) => {
        event.stopPropagation();
        event.dataTransfer.effectAllowed = "copy";
        writeImageFileDragData(event.dataTransfer, spec);
      }}
    >
      拖出复制
    </a>
  );
}
