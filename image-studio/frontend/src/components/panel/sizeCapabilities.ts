import { classifyImageModel } from "../../../../../shared/kernel/requestModel.js";
import {
  buildCustomAspectRatioId,
  reduceAspectRatio,
} from "../../lib/customAspectRatios.ts";
import type { APIMode, CustomAspectRatio, RequestPolicy, SizeValue } from "../../types/domain";

export type BuiltinAspectPreset = "auto" | "1:1" | "3:2" | "2:3" | "16:9" | "9:16";
export type AspectPreset = BuiltinAspectPreset | `custom:${string}`;
export type ResolutionPreset = "auto" | "1k" | "2k" | "4k";

export interface AspectPresetOption {
  value: AspectPreset;
  label: string;
  w: number;
  h: number;
  auto?: boolean;
  custom?: boolean;
}

export const ASPECT_PRESETS: AspectPresetOption[] = [
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

const BUILTIN_ASPECT_DIMENSIONS: Record<Exclude<BuiltinAspectPreset, "auto">, { width: number; height: number }> = {
  "1:1": { width: 1, height: 1 },
  "3:2": { width: 3, height: 2 },
  "2:3": { width: 2, height: 3 },
  "16:9": { width: 16, height: 9 },
  "9:16": { width: 9, height: 16 },
};

const SIZE_MATRIX: Record<Exclude<BuiltinAspectPreset, "auto">, Record<Exclude<ResolutionPreset, "auto">, SizeValue>> = {
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

const SIZE_TO_ASPECT: Record<string, BuiltinAspectPreset> = {
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
const DEFAULT_ASPECT_FROM_AUTO: Exclude<BuiltinAspectPreset, "auto"> = "1:1";
const DEFAULT_RESOLUTION_FROM_AUTO: Exclude<ResolutionPreset, "auto"> = "1k";
const CUSTOM_RESOLUTION_REFERENCES: Record<Exclude<ResolutionPreset, "auto">, { area: number; maxSide: number }> = {
  "1k": { area: 1536 * 1024, maxSide: 1536 },
  "2k": { area: 2048 * 1360, maxSide: 2048 },
  "4k": { area: 3840 * 2160, maxSide: 3840 },
};
const CUSTOM_ASPECT_TOLERANCE = 0.035;
const BUILTIN_ASPECT_TOLERANCE = 0.08;

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

export function buildCustomAspectValue(id: string): AspectPreset {
  return `custom:${id}` as AspectPreset;
}

export function customAspectIdFromValue(aspect: AspectPreset): string | null {
  return aspect.startsWith("custom:") ? aspect.slice("custom:".length) : null;
}

export function aspectPresetLabel(aspect: AspectPreset, customRatios: CustomAspectRatio[] = []): string {
  if (aspect === "auto") return "Auto";
  const customId = customAspectIdFromValue(aspect);
  if (customId) {
    return customRatios.find((item) => item.id === customId)?.label ?? customId;
  }
  return ASPECT_PRESETS.find((item) => item.value === aspect)?.label ?? aspect;
}

export function listAspectPresetOptions(customRatios: CustomAspectRatio[] = []): AspectPresetOption[] {
  return [
    ...ASPECT_PRESETS,
    ...customRatios.map((ratio) => {
      const shape = aspectShapeFromRatio(ratio.width, ratio.height);
      return {
        value: buildCustomAspectValue(ratio.id),
        label: ratio.label,
        w: shape.w,
        h: shape.h,
        custom: true,
      };
    }),
  ];
}

export function deriveAspectPreset(size: SizeValue, customRatios: CustomAspectRatio[] = []): AspectPreset {
  if (size === "auto") return "auto";
  const exact = SIZE_TO_ASPECT[size];
  if (exact) return exact;
  const parsed = parseSizeValue(size);
  if (!parsed) return DEFAULT_ASPECT_FROM_AUTO;
  const customMatch = findMatchingCustomAspect(parsed.width, parsed.height, customRatios);
  if (customMatch) return buildCustomAspectValue(customMatch.id);
  const builtinMatch = nearestBuiltinAspect(parsed.width, parsed.height);
  return builtinMatch.distance <= BUILTIN_ASPECT_TOLERANCE ? builtinMatch.value : DEFAULT_ASPECT_FROM_AUTO;
}

export function deriveResolutionPreset(size: SizeValue): ResolutionPreset {
  if (size === "auto") return "auto";
  const exact = SIZE_TO_RESOLUTION[size];
  if (exact) return exact;
  const parsed = parseSizeValue(size);
  if (!parsed) return DEFAULT_RESOLUTION_FROM_AUTO;
  const area = parsed.width * parsed.height;
  let best: ResolutionPreset = DEFAULT_RESOLUTION_FROM_AUTO;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (const [resolution, reference] of Object.entries(CUSTOM_RESOLUTION_REFERENCES) as Array<[Exclude<ResolutionPreset, "auto">, { area: number }]>) {
    const distance = Math.abs(area - reference.area);
    if (distance < bestDistance) {
      bestDistance = distance;
      best = resolution;
    }
  }
  return best;
}

export function buildSizeSelection(
  aspect: AspectPreset,
  resolution: ResolutionPreset,
  input: {
    apiMode: APIMode;
    requestPolicy: RequestPolicy;
    imageModelID?: string;
  },
  customRatios: CustomAspectRatio[] = [],
): SizeValue {
  if (aspect === "auto" || resolution === "auto") {
    return "auto";
  }
  const normalizedResolution = normalizeResolutionSelection(resolution, input);
  if (normalizedResolution === "auto") {
    return "auto";
  }
  const custom = customAspectIdFromValue(aspect);
  if (custom) {
    const ratio = customRatios.find((item) => item.id === custom);
    return ratio ? buildCustomSizeSelection(ratio, normalizedResolution) : SIZE_MATRIX[DEFAULT_ASPECT_FROM_AUTO][normalizedResolution];
  }
  return SIZE_MATRIX[aspect as Exclude<BuiltinAspectPreset, "auto">][normalizedResolution];
}

export function buildAspectSizeSelection(
  aspect: AspectPreset,
  currentResolution: ResolutionPreset,
  input: {
    apiMode: APIMode;
    requestPolicy: RequestPolicy;
    imageModelID?: string;
  },
  customRatios: CustomAspectRatio[] = [],
): SizeValue {
  if (aspect === "auto") return "auto";
  const normalizedResolution = normalizeResolutionSelection(currentResolution, input);
  return buildSizeSelection(
    aspect,
    normalizedResolution === "auto" ? DEFAULT_RESOLUTION_FROM_AUTO : normalizedResolution,
    input,
    customRatios,
  );
}

export function buildResolutionSizeSelection(
  currentAspect: AspectPreset,
  resolution: ResolutionPreset,
  input: {
    apiMode: APIMode;
    requestPolicy: RequestPolicy;
    imageModelID?: string;
  },
  customRatios: CustomAspectRatio[] = [],
): SizeValue {
  if (resolution === "auto") return "auto";
  return buildSizeSelection(
    currentAspect === "auto" ? DEFAULT_ASPECT_FROM_AUTO : currentAspect,
    normalizeResolutionSelection(resolution, input),
    input,
    customRatios,
  );
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
  return allowed.has(resolution) ? resolution : DEFAULT_RESOLUTION_FROM_AUTO;
}

export function normalizeSizeSelection(
  size: SizeValue,
  input: {
    apiMode: APIMode;
    requestPolicy: RequestPolicy;
    imageModelID?: string;
  },
  customRatios: CustomAspectRatio[] = [],
): SizeValue {
  const aspect = deriveAspectPreset(size, customRatios);
  const resolution = deriveResolutionPreset(size);
  return buildSizeSelection(aspect, resolution, input, customRatios);
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

function buildCustomSizeSelection(
  ratio: CustomAspectRatio,
  resolution: Exclude<ResolutionPreset, "auto">,
): SizeValue {
  const reference = CUSTOM_RESOLUTION_REFERENCES[resolution];
  const aspect = ratio.width / ratio.height;
  let width = Math.sqrt(reference.area * aspect);
  let height = Math.sqrt(reference.area / aspect);
  const maxSide = Math.max(width, height);
  if (maxSide > reference.maxSide) {
    const scale = reference.maxSide / maxSide;
    width *= scale;
    height *= scale;
  }
  return `${roundDimension(width)}x${roundDimension(height)}` as SizeValue;
}

function aspectShapeFromRatio(width: number, height: number): { w: number; h: number } {
  const maxWidth = 24;
  const maxHeight = 22;
  const safeWidth = Math.max(1, width);
  const safeHeight = Math.max(1, height);
  const scale = Math.min(maxWidth / safeWidth, maxHeight / safeHeight);
  return {
    w: Math.max(10, Math.round(safeWidth * scale)),
    h: Math.max(10, Math.round(safeHeight * scale)),
  };
}

function parseSizeValue(size: SizeValue): { width: number; height: number } | null {
  const match = /^(\d+)x(\d+)$/.exec(size);
  if (!match) return null;
  const width = Number(match[1]);
  const height = Number(match[2]);
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
    return null;
  }
  return { width, height };
}

function findMatchingCustomAspect(
  width: number,
  height: number,
  customRatios: CustomAspectRatio[],
): CustomAspectRatio | null {
  let best: CustomAspectRatio | null = null;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (const ratio of customRatios) {
    const distance = aspectRatioDistance(width, height, ratio.width, ratio.height);
    if (distance < bestDistance) {
      best = ratio;
      bestDistance = distance;
    }
  }
  return best && bestDistance <= CUSTOM_ASPECT_TOLERANCE ? best : null;
}

function nearestBuiltinAspect(width: number, height: number): {
  value: Exclude<BuiltinAspectPreset, "auto">;
  distance: number;
} {
  let bestValue: Exclude<BuiltinAspectPreset, "auto"> = DEFAULT_ASPECT_FROM_AUTO;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (const [value, dims] of Object.entries(BUILTIN_ASPECT_DIMENSIONS) as Array<[Exclude<BuiltinAspectPreset, "auto">, { width: number; height: number }]>) {
    const distance = aspectRatioDistance(width, height, dims.width, dims.height);
    if (distance < bestDistance) {
      bestValue = value;
      bestDistance = distance;
    }
  }
  return { value: bestValue, distance: bestDistance };
}

function aspectRatioDistance(width: number, height: number, targetWidth: number, targetHeight: number): number {
  const left = width / height;
  const right = targetWidth / targetHeight;
  return Math.abs(left - right) / right;
}

function roundDimension(value: number): number {
  return Math.max(64, Math.round(value / 8) * 8);
}

export function builtInAspectId(width: number, height: number): string {
  return buildCustomAspectRatioId(width, height);
}

export function isBuiltInAspectRatio(width: number, height: number): boolean {
  const id = buildCustomAspectRatioId(width, height);
  return id === "1:1" || id === "3:2" || id === "2:3" || id === "16:9" || id === "9:16";
}

export function reduceAspectRatioLabel(width: number, height: number): string {
  const reduced = reduceAspectRatio(width, height);
  return reduced.width > 0 && reduced.height > 0 ? `${reduced.width}:${reduced.height}` : "";
}
