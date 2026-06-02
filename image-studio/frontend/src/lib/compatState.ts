import { LoadCompatibilityState, SaveCompatibilityState } from "../platform/runtime/host";
import type {
  CustomAspectRatio,
  HistoryItem,
  KernelRuntimeMode,
  OutputFormatValue,
  Preset,
  ProxyMode,
  ThemeMode,
  UpstreamProfile,
} from "../types/domain";
import { ACTIVE_PROFILE_LS_KEY, PROFILES_LS_KEY, tryParseProfile } from "./profiles";
import { normalizeProxyMode, persistProxyConfig } from "./proxy";
import {
  normalizeCustomAspectRatios,
  persistCustomAspectRatios,
} from "./customAspectRatios.ts";
import {
  loadTrustedOutputRoots,
  persistHistoryFullImage,
  persistHistoryItem,
  pruneHistoryStorage,
} from "./storage";

const SCHEMA_VERSION = 1;
const MARKER_KEY = "gptcodex.compatStateUpdatedAt";

export type CompatibilityState = {
  schemaVersion: number;
  client?: string;
  updatedAt: number;
  settings: {
    proxyMode?: ProxyMode;
    proxyURL?: string;
    theme?: ThemeMode;
    fontScale?: number;
    outputFormat?: OutputFormatValue;
    outputDir?: string;
    promptHistory?: string[];
    presets?: Preset[];
    customAspectRatios?: CustomAspectRatio[];
    kernelRuntimeMode?: KernelRuntimeMode;
    trustedOutputRoots?: string[];
    savePromptSuppressed?: boolean;
  };
  profiles: UpstreamProfile[];
  activeProfileId: string;
  history: HistoryItem[];
  historyFull?: Array<{ id: string; imageB64: string }>;
};

export type CompatibilityExportInput = {
  history: HistoryItem[];
  profiles: UpstreamProfile[];
  activeProfileId: string;
  proxyMode: ProxyMode;
  proxyURL: string;
  theme: ThemeMode;
  fontScale: number;
  outputFormat: OutputFormatValue;
  promptHistory: string[];
  presets: Preset[];
  customAspectRatios: CustomAspectRatio[];
  kernelRuntimeMode: KernelRuntimeMode;
};

let exportTimer: ReturnType<typeof setTimeout> | null = null;

export async function importCompatibilityStateIfNewer(): Promise<boolean> {
  const state = normalizeCompatibilityState(await LoadCompatibilityState());
  if (!state || state.updatedAt <= 0) return false;
  if (state.updatedAt <= readLocalMarker()) return false;

  applyCompatibilityLocalStorage(state);
  await persistCompatibilityHistory(state);
  writeLocalMarker(state.updatedAt);
  return true;
}

export function scheduleCompatibilityExport(input: CompatibilityExportInput): void {
  if (exportTimer) clearTimeout(exportTimer);
  const snapshot = cloneExportInput(input);
  exportTimer = setTimeout(() => {
    exportTimer = null;
    void exportCompatibilityStateNow(snapshot).catch((error) => {
      if (typeof console !== "undefined") console.warn("compat export failed", error);
    });
  }, 250);
}

export async function exportCompatibilityStateNow(input: CompatibilityExportInput): Promise<void> {
  const state = buildCompatibilityState(input);
  await SaveCompatibilityState(state as unknown as Record<string, unknown>);
  writeLocalMarker(state.updatedAt);
}

export function compatibilityExportFingerprint(input: CompatibilityExportInput): string {
  return JSON.stringify({
    profiles: input.profiles,
    activeProfileId: input.activeProfileId,
    proxyMode: input.proxyMode,
    proxyURL: input.proxyURL,
    theme: input.theme,
    fontScale: input.fontScale,
    outputFormat: input.outputFormat,
    promptHistory: input.promptHistory,
    presets: input.presets,
    customAspectRatios: input.customAspectRatios,
    kernelRuntimeMode: input.kernelRuntimeMode,
    outputDir: readLocalStorageString("gptcodex.outputDir"),
    trustedOutputRoots: loadTrustedOutputRoots(),
    savePromptSuppressed: readLocalStorageString("gptcodex.savePromptSuppressed") === "1",
    history: input.history.map(historyFingerprint),
  });
}

function buildCompatibilityState(input: CompatibilityExportInput): CompatibilityState {
  const history = input.history.map(toSerializableHistoryItem).filter((item): item is HistoryItem => item !== null);
  return {
    schemaVersion: SCHEMA_VERSION,
    client: "webview2",
    updatedAt: Date.now(),
    settings: {
      proxyMode: normalizeProxyMode(input.proxyMode),
      proxyURL: input.proxyURL.trim(),
      theme: normalizeTheme(input.theme),
      fontScale: normalizeFontScale(input.fontScale),
      outputFormat: normalizeOutputFormat(input.outputFormat),
      outputDir: readLocalStorageString("gptcodex.outputDir"),
      promptHistory: cleanStringList(input.promptHistory, 50),
      presets: normalizePresets(input.presets),
      customAspectRatios: normalizeCustomAspectRatios(input.customAspectRatios),
      kernelRuntimeMode: normalizeKernelRuntimeMode(input.kernelRuntimeMode),
      trustedOutputRoots: loadTrustedOutputRoots(),
      savePromptSuppressed: readLocalStorageString("gptcodex.savePromptSuppressed") === "1",
    },
    profiles: normalizeProfiles(input.profiles),
    activeProfileId: input.activeProfileId || "",
    history,
    historyFull: history
      .filter((item) => typeof item.imageB64 === "string" && item.imageB64.trim())
      .map((item) => ({ id: item.id, imageB64: item.imageB64 as string })),
  };
}

function applyCompatibilityLocalStorage(state: CompatibilityState): void {
  const profiles = normalizeProfiles(state.profiles);
  writeLocalStorageJSON(PROFILES_LS_KEY, profiles);
  if (state.activeProfileId) writeLocalStorageString(ACTIVE_PROFILE_LS_KEY, state.activeProfileId);
  else removeLocalStorage(ACTIVE_PROFILE_LS_KEY);

  const settings = state.settings ?? {};
  if (settings.proxyMode) persistProxyConfig(normalizeProxyMode(settings.proxyMode), settings.proxyURL ?? "");
  if (settings.theme) writeLocalStorageString("gptcodex.theme", normalizeTheme(settings.theme));
  if (typeof settings.fontScale === "number") writeLocalStorageString("gptcodex.fontScale", String(normalizeFontScale(settings.fontScale)));
  if (settings.outputFormat) writeLocalStorageString("gptcodex.outputFormat", normalizeOutputFormat(settings.outputFormat));
  if (settings.outputDir?.trim()) writeLocalStorageString("gptcodex.outputDir", settings.outputDir.trim());
  else removeLocalStorage("gptcodex.outputDir");
  if (settings.kernelRuntimeMode) writeLocalStorageString("gptcodex.kernelRuntimeMode", normalizeKernelRuntimeMode(settings.kernelRuntimeMode));
  writeLocalStorageJSON("gptcodex.promptHistory", cleanStringList(settings.promptHistory ?? [], 50));
  writeLocalStorageJSON("gptcodex.presets", normalizePresets(settings.presets ?? []));
  persistCustomAspectRatios(normalizeCustomAspectRatios(settings.customAspectRatios ?? []));
  writeLocalStorageJSON("gptcodex.trustedOutputRoots", cleanStringList(settings.trustedOutputRoots ?? [], 100));
  if (settings.savePromptSuppressed) writeLocalStorageString("gptcodex.savePromptSuppressed", "1");
  else removeLocalStorage("gptcodex.savePromptSuppressed");
}

async function persistCompatibilityHistory(state: CompatibilityState): Promise<void> {
  const items = state.history.map(toSerializableHistoryItem).filter((item): item is HistoryItem => item !== null);
  for (const item of items) {
    await persistHistoryItem(item).catch(() => undefined);
    if (typeof item.imageB64 === "string" && item.imageB64.trim()) {
      await persistHistoryFullImage(item.id, item.imageB64).catch(() => undefined);
    }
  }
  for (const full of state.historyFull ?? []) {
    if (!full?.id || !full.imageB64?.trim()) continue;
    await persistHistoryFullImage(full.id, full.imageB64).catch(() => undefined);
  }
  await pruneHistoryStorage(items.map((item) => item.id)).catch(() => undefined);
}

function normalizeCompatibilityState(raw: unknown): CompatibilityState | null {
  if (!raw || typeof raw !== "object") return null;
  const source = raw as Record<string, any>;
  return {
    schemaVersion: typeof source.schemaVersion === "number" ? source.schemaVersion : SCHEMA_VERSION,
    client: typeof source.client === "string" ? source.client : undefined,
    updatedAt: typeof source.updatedAt === "number" ? source.updatedAt : 0,
    settings: normalizeSettings(source.settings),
    profiles: normalizeProfiles(Array.isArray(source.profiles) ? source.profiles : []),
    activeProfileId: typeof source.activeProfileId === "string" ? source.activeProfileId : "",
    history: (Array.isArray(source.history) ? source.history : [])
      .map(toSerializableHistoryItem)
      .filter((item): item is HistoryItem => item !== null),
    historyFull: (Array.isArray(source.historyFull) ? source.historyFull : [])
      .map((item: any) => ({
        id: typeof item?.id === "string" ? item.id : "",
        imageB64: typeof item?.imageB64 === "string" ? item.imageB64 : "",
      }))
      .filter((item: { id: string; imageB64: string }) => item.id && item.imageB64),
  };
}

function normalizeSettings(raw: unknown): CompatibilityState["settings"] {
  const source = raw && typeof raw === "object" ? raw as Record<string, any> : {};
  return {
    proxyMode: normalizeProxyMode(source.proxyMode),
    proxyURL: typeof source.proxyURL === "string" ? source.proxyURL : "",
    theme: normalizeTheme(source.theme),
    fontScale: normalizeFontScale(source.fontScale),
    outputFormat: normalizeOutputFormat(source.outputFormat),
    outputDir: typeof source.outputDir === "string" ? source.outputDir : "",
    promptHistory: cleanStringList(source.promptHistory ?? [], 50),
    presets: normalizePresets(source.presets ?? []),
    customAspectRatios: normalizeCustomAspectRatios(source.customAspectRatios ?? []),
    kernelRuntimeMode: normalizeKernelRuntimeMode(source.kernelRuntimeMode),
    trustedOutputRoots: cleanStringList(source.trustedOutputRoots ?? [], 100),
    savePromptSuppressed: source.savePromptSuppressed === true,
  };
}

function normalizeProfiles(raw: unknown[]): UpstreamProfile[] {
  return raw.map((item) => tryParseProfile(item)).filter((profile): profile is UpstreamProfile => profile !== null);
}

function toSerializableHistoryItem(raw: unknown): HistoryItem | null {
  if (!raw || typeof raw !== "object") return null;
  const item = raw as Record<string, any>;
  const id = typeof item.id === "string" ? item.id.trim() : "";
  const prompt = typeof item.prompt === "string" ? item.prompt : "";
  const createdAt = typeof item.createdAt === "number" ? item.createdAt : 0;
  if (!id || !createdAt) return null;
  return {
    id,
    imageId: stringOrUndefined(item.imageId),
    previewUrl: stringOrUndefined(item.previewUrl),
    fullUrl: stringOrUndefined(item.fullUrl),
    thumbPath: stringOrUndefined(item.thumbPath),
    previewWidth: numberOrUndefined(item.previewWidth),
    previewHeight: numberOrUndefined(item.previewHeight),
    imageB64: stringOrUndefined(item.imageB64),
    imageBlob: null,
    previewBlob: null,
    previewOnly: item.previewOnly === true,
    prompt,
    revisedPrompt: stringOrUndefined(item.revisedPrompt),
    mode: item.mode === "edit" ? "edit" : "generate",
    size: normalizeSize(item.size),
    quality: normalizeQuality(item.quality),
    outputFormat: normalizeOutputFormat(item.outputFormat),
    parentId: stringOrUndefined(item.parentId),
    createdAt,
    seed: numberOrUndefined(item.seed),
    negativePrompt: stringOrUndefined(item.negativePrompt),
    styleTag: stringOrUndefined(item.styleTag),
    batchIndex: numberOrUndefined(item.batchIndex),
    elapsedSec: numberOrUndefined(item.elapsedSec),
    savedPath: stringOrUndefined(item.savedPath),
    rawPath: stringOrUndefined(item.rawPath),
  };
}

function historyFingerprint(item: HistoryItem) {
  return {
    id: item.id,
    imageId: item.imageId,
    savedPath: item.savedPath,
    thumbPath: item.thumbPath,
    rawPath: item.rawPath,
    prompt: item.prompt,
    revisedPrompt: item.revisedPrompt,
    mode: item.mode,
    size: item.size,
    quality: item.quality,
    outputFormat: item.outputFormat,
    createdAt: item.createdAt,
    seed: item.seed,
    negativePrompt: item.negativePrompt,
    styleTag: item.styleTag,
    batchIndex: item.batchIndex,
    elapsedSec: item.elapsedSec,
    imageB64: item.imageB64,
  };
}

function cloneExportInput(input: CompatibilityExportInput): CompatibilityExportInput {
  return {
    history: input.history.map((item) => ({ ...item, imageBlob: null, previewBlob: null })),
    profiles: input.profiles.map((profile) => ({ ...profile })),
    activeProfileId: input.activeProfileId,
    proxyMode: input.proxyMode,
    proxyURL: input.proxyURL,
    theme: input.theme,
    fontScale: input.fontScale,
    outputFormat: input.outputFormat,
    promptHistory: [...input.promptHistory],
    presets: input.presets.map((preset) => ({ ...preset })),
    customAspectRatios: input.customAspectRatios.map((ratio) => ({ ...ratio })),
    kernelRuntimeMode: input.kernelRuntimeMode,
  };
}

function normalizePresets(raw: unknown): Preset[] {
  if (!Array.isArray(raw)) return [];
  const out: Preset[] = [];
  for (const item of raw) {
    if (!item || typeof item !== "object") continue;
    const source = item as Record<string, any>;
    const id = typeof source.id === "string" ? source.id : "";
    const name = typeof source.name === "string" ? source.name.trim() : "";
    if (!id || !name) continue;
    out.push({
      id,
      name,
      size: normalizeSize(source.size),
      quality: normalizeQuality(source.quality),
      outputFormat: normalizeOutputFormat(source.outputFormat),
      negativePrompt: typeof source.negativePrompt === "string" ? source.negativePrompt : "",
      kernelRuntimeMode: normalizeKernelRuntimeMode(source.kernelRuntimeMode),
      batchCount: normalizeBatchCount(source.batchCount),
    });
  }
  return out;
}

function normalizeTheme(value: unknown): ThemeMode {
  return value === "light" || value === "dark" || value === "system" ? value : "system";
}

function normalizeKernelRuntimeMode(value: unknown): KernelRuntimeMode {
  return value === "local" || value === "remote" || value === "auto" ? value : "auto";
}

function normalizeOutputFormat(value: unknown): OutputFormatValue {
  return value === "jpeg" || value === "webp" || value === "png" ? value : "png";
}

function normalizeSize(value: unknown): HistoryItem["size"] {
  return typeof value === "string" && value.trim() ? value.trim() as HistoryItem["size"] : "1024x1024";
}

function normalizeQuality(value: unknown): HistoryItem["quality"] {
  return value === "auto" || value === "high" || value === "low" || value === "medium" ? value : "medium";
}

function normalizeFontScale(value: unknown): number {
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) && n > 0.5 && n < 2 ? n : 1;
}

function normalizeBatchCount(value: unknown): number {
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) && n > 0 ? Math.min(9, Math.floor(n)) : 1;
}

function cleanStringList(raw: unknown, max: number): string[] {
  if (!Array.isArray(raw)) return [];
  return Array.from(new Set(
    raw.map((item) => typeof item === "string" ? item.trim() : "").filter(Boolean),
  )).slice(0, max);
}

function stringOrUndefined(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function numberOrUndefined(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function readLocalMarker(): number {
  const raw = readLocalStorageString(MARKER_KEY);
  const value = Number(raw);
  return Number.isFinite(value) ? value : 0;
}

function writeLocalMarker(value: number): void {
  writeLocalStorageString(MARKER_KEY, String(Math.max(0, Math.floor(value))));
}

function readLocalStorageString(key: string): string {
  try { return localStorage.getItem(key) ?? ""; } catch { return ""; }
}

function writeLocalStorageString(key: string, value: string): void {
  try { localStorage.setItem(key, value); } catch {}
}

function writeLocalStorageJSON(key: string, value: unknown): void {
  try { localStorage.setItem(key, JSON.stringify(value)); } catch {}
}

function removeLocalStorage(key: string): void {
  try { localStorage.removeItem(key); } catch {}
}
