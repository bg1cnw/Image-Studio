import type { APIMode, RequestPolicy, ReasoningEffortValue, UpstreamProfile } from "../types/domain";

export type UpstreamConfigExportProfile = {
  id: string;
  name: string;
  apiMode: APIMode;
  requestPolicy: RequestPolicy;
  imagesNewAPICompat: boolean;
  baseURL: string;
  textModelID: string;
  imageModelID: string;
  reasoningEffort: ReasoningEffortValue;
  concurrencyLimit: number;
  fallbackProfileId?: string;
  createdAt: number;
  lastUsedAt?: number;
  apiKey?: string;
};

export type UpstreamConfigExportFile = {
  version: 1;
  exportedAt: string;
  activeProfileId: string;
  profiles: UpstreamConfigExportProfile[];
};

export type ParsedUpstreamConfigImport = {
  activeProfileId: string;
  profiles: UpstreamConfigExportProfile[];
};

function normalizeAPIMode(value: unknown): APIMode {
  return value === "images" ? "images" : "responses";
}

function normalizeRequestPolicy(value: unknown): RequestPolicy {
  return value === "compat" ? "compat" : "openai";
}

function normalizeReasoningEffort(value: unknown): ReasoningEffortValue {
  return value === "low" || value === "medium" || value === "high" || value === "xhigh"
    ? value
    : "xhigh";
}

function normalizeConcurrencyLimit(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value) || value < 0) return 0;
  return Math.floor(value);
}

function parseExportProfile(raw: unknown): UpstreamConfigExportProfile | null {
  if (!raw || typeof raw !== "object") return null;
  const source = raw as Record<string, unknown>;
  const id = typeof source.id === "string" ? source.id.trim() : "";
  const name = typeof source.name === "string" ? source.name.trim() : "";
  if (!id || !name) return null;
  return {
    id,
    name,
    apiMode: normalizeAPIMode(source.apiMode),
    requestPolicy: normalizeRequestPolicy(source.requestPolicy),
    imagesNewAPICompat: source.imagesNewAPICompat === true,
    baseURL: typeof source.baseURL === "string" ? source.baseURL.trim() : "",
    textModelID: typeof source.textModelID === "string" ? source.textModelID.trim() : "",
    imageModelID: typeof source.imageModelID === "string" ? source.imageModelID.trim() : "",
    reasoningEffort: normalizeReasoningEffort(source.reasoningEffort),
    concurrencyLimit: normalizeConcurrencyLimit(source.concurrencyLimit),
    fallbackProfileId: typeof source.fallbackProfileId === "string" ? source.fallbackProfileId.trim() || undefined : undefined,
    createdAt: typeof source.createdAt === "number" && Number.isFinite(source.createdAt) ? source.createdAt : Date.now(),
    lastUsedAt: typeof source.lastUsedAt === "number" && Number.isFinite(source.lastUsedAt) ? source.lastUsedAt : undefined,
    apiKey: typeof source.apiKey === "string" ? source.apiKey.trim() : "",
  };
}

export function buildUpstreamConfigExportFile(
  profiles: UpstreamProfile[],
  activeProfileId: string,
  apiKeysById: Record<string, string>,
): UpstreamConfigExportFile {
  return {
    version: 1,
    exportedAt: new Date().toISOString(),
    activeProfileId,
    profiles: profiles.map((profile) => ({
      ...profile,
      imagesNewAPICompat: profile.imagesNewAPICompat === true,
      apiKey: (apiKeysById[profile.id] ?? "").trim(),
    })),
  };
}

export function parseUpstreamConfigImportFile(rawJSON: string): ParsedUpstreamConfigImport {
  const parsed = JSON.parse(rawJSON) as Record<string, unknown>;
  const profiles = Array.isArray(parsed?.profiles)
    ? parsed.profiles.map(parseExportProfile).filter((item): item is UpstreamConfigExportProfile => item !== null)
    : [];
  if (profiles.length === 0) {
    throw new Error("文件里没有可导入的上游配置");
  }
  const activeProfileId = typeof parsed?.activeProfileId === "string" ? parsed.activeProfileId.trim() : "";
  return {
    activeProfileId,
    profiles,
  };
}
