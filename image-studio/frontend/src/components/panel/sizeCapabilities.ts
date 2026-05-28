import { classifyImageModel } from "../../../../../shared/kernel/requestModel.js";
import type { APIMode, RequestPolicy, SizeValue } from "../../types/domain";

export type AspectPreset = "auto" | "1:1" | "3:2" | "2:3" | "16:9" | "9:16";
export type ResolutionPreset = "auto" | "1k" | "2k" | "4k";

export const ASPECT_PRESETS: Array<{ value: AspectPreset; label: string; w: number; h: number; auto?: boolean }> = [
  { value: "auto", label: "Auto", w: 18, h: 18, auto: true },
  { value: "1:1", label: "1:1", w: 18, h: 18 },
  { value: "3:2", label: "3:2", w: 22, h: 14 },
  { value: "2:3", label: "2:3", w: 14, h: 20 },
  { value: "16:9", label: "16:9", w: 24, h: 13 },
  { value: "9:16", label: "9:16", w: 12, h: 22 },
];

export const RESOLUTION_PRESETS: Array<{ value: ResolutionPreset; label: string }> = [
  { value: "auto", label: "自动" },
  { value: "1k", label: "1K" },
  { value: "2k", label: "2K" },
  { value: "4k", label: "4K" },
];

const SIZE_MATRIX: Record<Exclude<AspectPreset, "auto">, Record<Exclude<ResolutionPreset, "auto">, SizeValue>> = {
  "1:1": {
    "1k": "1024x1024",
    "2k": "2048x2048",
    "4k": "2880x2880",
  },
  "3:2": {
    "1k": "1536x1024",
    "2k": "2048x1360",
    "4k": "3456x2304",
  },
  "2:3": {
    "1k": "1024x1536",
    "2k": "1360x2048",
    "4k": "2304x3456",
  },
  "16:9": {
    "1k": "1536x864",
    "2k": "2048x1152",
    "4k": "3840x2160",
  },
  "9:16": {
    "1k": "864x1536",
    "2k": "1152x2048",
    "4k": "2160x3840",
  },
};

const SIZE_TO_ASPECT: Record<string, AspectPreset> = {
  auto: "auto",
  "1024x1024": "1:1",
  "2048x2048": "1:1",
  "2880x2880": "1:1",
  "1536x1024": "3:2",
  "2048x1360": "3:2",
  "3456x2304": "3:2",
  "1024x1536": "2:3",
  "1360x2048": "2:3",
  "2304x3456": "2:3",
  "1536x864": "16:9",
  "2048x1152": "16:9",
  "3840x2160": "16:9",
  "864x1536": "9:16",
  "1152x2048": "9:16",
  "2160x3840": "9:16",
};

const SIZE_TO_RESOLUTION: Record<string, ResolutionPreset> = {
  auto: "auto",
  "1024x1024": "1k",
  "1536x1024": "1k",
  "1024x1536": "1k",
  "1536x864": "1k",
  "864x1536": "1k",
  "2048x2048": "2k",
  "2048x1360": "2k",
  "1360x2048": "2k",
  "2048x1152": "2k",
  "1152x2048": "2k",
  "2880x2880": "4k",
  "3456x2304": "4k",
  "2304x3456": "4k",
  "3840x2160": "4k",
  "2160x3840": "4k",
};

const LARGE_RESOLUTION_PRESETS = new Set<ResolutionPreset>(["2k", "4k"]);

export function supportsExplicitLargeSizes({
  apiMode,
  requestPolicy,
  imageModelID,
}: {
  apiMode: APIMode;
  requestPolicy: RequestPolicy;
  imageModelID?: string;
}): boolean {
  const family = classifyImageModel(imageModelID || "");
  if (apiMode === "images") {
    return family === "gpt-image" || family === "dalle3";
  }
  if (family === "gpt-image") {
    return true;
  }
  return requestPolicy === "compat";
}

export function availableResolutionPresets(input: {
  apiMode: APIMode;
  requestPolicy: RequestPolicy;
  imageModelID?: string;
}): ResolutionPreset[] {
  const all: ResolutionPreset[] = ["auto", "1k", "2k", "4k"];
  if (supportsExplicitLargeSizes(input)) {
    return all;
  }
  return all.filter((value) => !LARGE_RESOLUTION_PRESETS.has(value));
}

export function deriveAspectPreset(size: SizeValue): AspectPreset {
  return SIZE_TO_ASPECT[size] ?? "1:1";
}

export function deriveResolutionPreset(size: SizeValue): ResolutionPreset {
  return SIZE_TO_RESOLUTION[size] ?? "1k";
}

export function buildSizeSelection(
  aspect: AspectPreset,
  resolution: ResolutionPreset,
  input: {
    apiMode: APIMode;
    requestPolicy: RequestPolicy;
    imageModelID?: string;
  },
): SizeValue {
  if (aspect === "auto" || resolution === "auto") {
    return "auto";
  }
  const normalizedResolution = normalizeResolutionSelection(resolution, input);
  if (normalizedResolution === "auto") {
    return "auto";
  }
  return SIZE_MATRIX[aspect][normalizedResolution];
}

export function normalizeResolutionSelection(
  resolution: ResolutionPreset,
  input: {
    apiMode: APIMode;
    requestPolicy: RequestPolicy;
    imageModelID?: string;
  },
): ResolutionPreset {
  const allowed = new Set(availableResolutionPresets(input));
  return allowed.has(resolution) ? resolution : "1k";
}

export function normalizeSizeSelection(
  size: SizeValue,
  input: {
    apiMode: APIMode;
    requestPolicy: RequestPolicy;
    imageModelID?: string;
  },
): SizeValue {
  const aspect = deriveAspectPreset(size);
  const resolution = deriveResolutionPreset(size);
  return buildSizeSelection(aspect, resolution, input);
}

export function sizeCapabilityHint(input: {
  apiMode: APIMode;
  requestPolicy: RequestPolicy;
  imageModelID?: string;
}): string {
  if (supportsExplicitLargeSizes(input)) {
    return "";
  }
  return "当前链路只保证基础尺寸稳定可用。";
}
