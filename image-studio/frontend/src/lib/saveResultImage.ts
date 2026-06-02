import type { HistoryItem, OutputFormatValue } from "../types/domain";
import { SaveImageAs, SaveImagePathAs, SaveImagePathToDir, SaveImageToDir } from "../platform/runtime/host";
import { saveImageForPlatform } from "../platform/android/bridge";

function extensionForFormat(format?: OutputFormatValue): string {
  if (format === "jpeg") return "jpg";
  if (format === "webp") return "webp";
  return "png";
}

function extensionFromPath(path?: string): string {
  const match = path?.match(/\.([a-z0-9]+)$/i);
  return match?.[1]?.toLowerCase() ?? "";
}

export function suggestedImageName(item: Pick<HistoryItem, "id" | "mode" | "outputFormat" | "savedPath">): string {
  const ext = extensionFromPath(item.savedPath) || extensionForFormat(item.outputFormat);
  return `image-${item.mode}-${item.id.slice(0, 8)}.${ext}`;
}

export async function saveHistoryItemAs(item: HistoryItem): Promise<string> {
  const suggested = suggestedImageName(item);
  if (item.savedPath) return SaveImagePathAs(item.savedPath, suggested);
  if (item.imageB64) return saveImageForPlatform(item.imageB64, suggested, SaveImageAs);
  throw new Error("当前图片没有可保存内容");
}

export async function saveHistoryItemToDirectory(item: HistoryItem, directory: string): Promise<string> {
  const suggested = suggestedImageName(item);
  if (item.savedPath) return SaveImagePathToDir(item.savedPath, directory, suggested);
  if (item.imageB64) return SaveImageToDir(item.imageB64, directory, suggested);
  throw new Error("当前图片没有可保存内容");
}
