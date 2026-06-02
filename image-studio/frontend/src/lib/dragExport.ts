import type { HistoryItem } from "../types/domain";
import { detectImageMimeTypeFromBase64, guessImageMimeTypeFromName } from "./images.ts";
import { suggestedImageName } from "./imageFileNames.ts";

export type DragExportSpec = {
  href: string;
  fileName: string;
  mimeType: string;
  downloadURL: string;
};

type DragExportHistoryItem = Pick<
  HistoryItem,
  "id" | "mode" | "outputFormat" | "savedPath" | "imageId" | "fullUrl" | "imageB64" | "previewOnly"
>;

function basename(path?: string): string {
  if (!path) return "";
  return path.split(/[\\/]/).pop() ?? "";
}

function mediaFullURL(imageId?: string): string {
  return imageId ? `/media/full/${imageId}` : "";
}

function isTransientPreview(item: DragExportHistoryItem): boolean {
  return !!item.previewOnly && item.id.startsWith("preview-");
}

function resolveAbsoluteURL(rawURL: string): string {
  if (typeof window === "undefined" || !window.location?.href) return rawURL;
  try {
    return new URL(rawURL, window.location.href).toString();
  } catch {
    return rawURL;
  }
}

export function buildHistoryItemDragExport(
  item: DragExportHistoryItem | null | undefined,
  sourceURL?: string | null,
): DragExportSpec | null {
  if (!item) return null;
  const fileName = basename(item.savedPath) || suggestedImageName(item);
  const preferredURL = sourceURL?.trim() || "";
  const fullURL = isTransientPreview(item) ? "" : (item.fullUrl || mediaFullURL(item.imageId));
  const rawURL = fullURL || preferredURL || (item.imageB64
    ? `data:${detectImageMimeTypeFromBase64(item.imageB64) || "image/png"};base64,${item.imageB64}`
    : "");
  if (!rawURL) return null;
  const href = resolveAbsoluteURL(rawURL);
  const mimeType = guessImageMimeTypeFromName(fileName)
    || (href.startsWith("data:image/")
      ? href.slice("data:".length, href.indexOf(";"))
      : "")
    || "application/octet-stream";
  return {
    href,
    fileName,
    mimeType,
    downloadURL: `${mimeType}:${fileName}:${href}`,
  };
}

export function writeImageFileDragData(
  dataTransfer: Pick<DataTransfer, "clearData" | "setData"> | null | undefined,
  spec: DragExportSpec,
): void {
  if (!dataTransfer) return;
  try {
    dataTransfer.clearData();
  } catch {
    // Some embedded webviews reject clearData during native drags.
  }
  for (const [format, value] of [
    ["DownloadURL", spec.downloadURL],
    ["text/uri-list", spec.href],
    ["text/plain", spec.href],
  ] as const) {
    try {
      dataTransfer.setData(format, value);
    } catch {
      // Keep writing the remaining formats so every webview gets its best shot.
    }
  }
}
