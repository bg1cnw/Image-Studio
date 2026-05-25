import { targetPlatform } from "./platform.ts";
import {
  probeUpstreamConnection,
  RemoteKernelError,
  runRemoteImageJob,
  optimizePromptRemote,
} from "./remoteKernel.ts";
import {
  hasAndroidInvokeBridge,
  invokeAndroidNative,
} from "./androidNativeInvoke.ts";
import {
  cropVirtualImage,
  flipVirtualImage,
  isVirtualPath,
  openImageDialogFallback,
  openVirtualPath,
  readVirtualImageAsBase64,
  readVirtualText,
  registerVirtualImage,
  registerVirtualText,
  rotateVirtualImage,
} from "./virtualHostStore.ts";

type AnyFn = (...args: any[]) => any;
type ServiceBinding = Record<string, AnyFn>;
type RuntimeBinding = {
  EventsOnMultiple?: (eventName: string, callback: (...args: any[]) => void, maxCallbacks?: number) => () => void;
  EventsOff?: (eventName: string, ...additionalEventNames: string[]) => void;
  WindowSetSystemDefaultTheme?: () => void;
  WindowSetLightTheme?: () => void;
  WindowSetDarkTheme?: () => void;
};

type GenerateOptionsLike = {
  apiKey: string;
  mode: string;
  prompt: string;
  size: string;
  quality: string;
  outputFormat: string;
  imagePaths: string[];
  imagePath: string;
  maskB64: string;
  seed: number;
  negativePrompt: string;
  baseURL: string;
  textModelID: string;
  imageModelID: string;
  transport: string;
  apiMode: string;
  noPromptRevision: boolean;
  concurrencyLimit?: number;
};

type PromptOptimizeOptionsLike = {
  apiKey: string;
  prompt: string;
  mode: string;
  baseURL: string;
  textModelID: string;
  imagePaths: string[];
  imagePath: string;
};

type JobStartedLike = { jobId: string };
type ImportedImageLike = { path: string; imageB64: string };
type SelectFileResponseLike = { path: string; size: number; imageB64?: string };

export type HostKind = "wails-desktop" | "android-shell" | "browser";

export type HostCapabilities = {
  localGeneration: boolean;
  promptOptimization: boolean;
  nativeFileDialogs: boolean;
  nativeImageTransforms: boolean;
  nativeHistoryFileIO: boolean;
  nativeOutputDirectoryPicker: boolean;
  secureCredentialStore: boolean;
};

export type KernelRuntimeMode = "auto" | "local" | "remote";

type BrowserWindow = Window & {
  go?: {
    backend?: {
      Service?: ServiceBinding;
    };
  };
  runtime?: RuntimeBinding;
  AndroidImageStudio?: {
    invoke?: (requestId: string, method: string, payloadJson: string) => void;
  };
  __imageStudioNativeResolve?: (requestId: string, payload: unknown) => void;
  __imageStudioNativeReject?: (requestId: string, message: string) => void;
};

const browserKeyPrefix = "image-studio.browser-key.";
const localEventListeners = new Map<string, Set<(...args: any[]) => void>>();
const remoteJobControllers = new Map<string, AbortController>();
let forcedKernelRuntimeMode: KernelRuntimeMode = "auto";

function getService(): ServiceBinding | null {
  if (typeof window === "undefined") return null;
  return (window as BrowserWindow).go?.backend?.Service ?? null;
}

function getRuntime(): RuntimeBinding | null {
  if (typeof window === "undefined") return null;
  return (window as BrowserWindow).runtime ?? null;
}

function getAndroidBridge() {
  if (typeof window === "undefined") return null;
  return (window as BrowserWindow).AndroidImageStudio ?? null;
}

function hasServiceMethod(name: string): boolean {
  return typeof getService()?.[name] === "function";
}

function canInvokeAndroidMethod(_name: string): boolean {
  return hasAndroidInvokeBridge();
}

function unsupportedMessage(method: string): string {
  const kind = detectHostKind();
  if (kind === "android-shell") {
    return `当前 Android shell 未提供 ${method} 对应的本地内核能力`;
  }
  if (kind === "browser") {
    return `当前浏览器预览环境未注入 ${method} 宿主能力`;
  }
  return `宿主未暴露 ${method} 能力`;
}

function invokeService<T>(method: string, ...args: unknown[]): Promise<T> {
  const fn = getService()?.[method];
  if (typeof fn !== "function") {
    return Promise.reject(new Error(unsupportedMessage(method)));
  }
  try {
    return Promise.resolve(fn(...args)) as Promise<T>;
  } catch (error) {
    return Promise.reject(error);
  }
}

function invokeAndroid<T>(method: string, ...args: unknown[]): Promise<T> {
  return invokeAndroidNative<T>(method, ...args).catch((error) => {
    if (String((error as any)?.message || "").includes("当前 Android shell 未提供")) {
      return Promise.reject(new Error(unsupportedMessage(method)));
    }
    return Promise.reject(error);
  });
}

function makeJobID(): string {
  try {
    if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
      return `job-${crypto.randomUUID()}`;
    }
  } catch {
    // ignore
  }
  return `job-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function emitLocalEvent(eventName: string, payload: unknown) {
  const bucket = localEventListeners.get(eventName);
  if (!bucket) return;
  for (const listener of Array.from(bucket)) {
    try {
      listener(payload);
    } catch {
      // ignore listener errors
    }
  }
}

function onLocalEvent(eventName: string, callback: (...args: any[]) => void) {
  const bucket = localEventListeners.get(eventName) ?? new Set<(...args: any[]) => void>();
  bucket.add(callback);
  localEventListeners.set(eventName, bucket);
  return () => {
    const existing = localEventListeners.get(eventName);
    if (!existing) return;
    existing.delete(callback);
    if (existing.size === 0) localEventListeners.delete(eventName);
  };
}

function clearLocalEvents(...eventNames: string[]) {
  for (const eventName of eventNames) {
    localEventListeners.delete(eventName);
  }
}

function saveByDownload(blob: Blob, suggestedName: string): string {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = suggestedName;
  document.body.appendChild(a);
  a.click();
  a.remove();
  globalThis.setTimeout(() => URL.revokeObjectURL(url), 15_000);
  return suggestedName;
}

function base64SizeBytes(b64: string): number {
  const clean = b64.replace(/=+$/, "");
  return Math.floor((clean.length * 3) / 4);
}

function browserStoredAPIKey(user: string): string {
  try {
    return localStorage.getItem(browserKeyPrefix + user) ?? "";
  } catch {
    return "";
  }
}

function setBrowserStoredAPIKey(user: string, value: string) {
  try {
    if (value.trim()) localStorage.setItem(browserKeyPrefix + user, value.trim());
    else localStorage.removeItem(browserKeyPrefix + user);
  } catch {
    // ignore
  }
}

function fileNameFromPath(path: string | undefined): string {
  if (!path) return "image.png";
  return path.split(/[\\/]/).pop() || "image.png";
}

async function materializeReadablePathAsVirtual(path: string): Promise<ImportedImageLike> {
  const imageB64 = await ReadImageAsBase64(path);
  return registerVirtualImage({
    imageB64,
    suggestedName: fileNameFromPath(path),
  });
}

async function importHistoryFallback(): Promise<string> {
  if (typeof document === "undefined") return "";
  return new Promise((resolve, reject) => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = "application/json,.json";
    input.style.position = "fixed";
    input.style.left = "-9999px";
    document.body.appendChild(input);
    const cleanup = () => input.remove();
    input.addEventListener("change", async () => {
      try {
        const file = input.files?.[0];
        if (!file) {
          cleanup();
          resolve("");
          return;
        }
        cleanup();
        resolve(await file.text());
      } catch (error) {
        cleanup();
        reject(error);
      }
    }, { once: true });
    input.click();
  });
}

async function startRemoteJob(options: GenerateOptionsLike): Promise<JobStartedLike> {
  const jobId = makeJobID();
  const controller = new AbortController();
  remoteJobControllers.set(jobId, controller);
  void (async () => {
    try {
      const result = await runRemoteImageJob({ payload: options }, {
        signal: controller.signal,
        onLog: (line) => emitLocalEvent(`log:${jobId}`, line),
        onProgress: (stage, elapsed, bytes) => emitLocalEvent(`progress:${jobId}`, { stage, elapsed, bytes }),
      });
      if (controller.signal.aborted) return;
      const saved = registerVirtualImage({
        imageB64: result.imageB64,
        suggestedName: `image-${options.mode || "generate"}.${options.outputFormat || "png"}`,
      });
      emitLocalEvent(`result:${jobId}`, {
        imageB64: result.imageB64,
        revisedPrompt: result.revisedPrompt,
        sourceEvent: result.sourceEvent,
        savedPath: saved.path,
        rawPath: result.rawPath,
        mode: result.mode,
        prompt: result.prompt,
      });
    } catch (error) {
      if (controller.signal.aborted) return;
      const typed = error instanceof RemoteKernelError
        ? error
        : new RemoteKernelError(String((error as any)?.message || error));
      emitLocalEvent(`error:${jobId}`, {
        message: typed.message,
        rawPath: typed.rawPath || null,
      });
    } finally {
      remoteJobControllers.delete(jobId);
    }
  })();
  return { jobId };
}

export function detectHostKind(): HostKind {
  if (targetPlatform === "android" || targetPlatform === "android-pad") {
    return canInvokeAndroidMethod("GetOutputDir") || hasServiceMethod("GetOutputDir") ? "android-shell" : "browser";
  }
  if (hasServiceMethod("Generate") && getRuntime()) return "wails-desktop";
  return "browser";
}

export function setKernelRuntimeMode(mode: KernelRuntimeMode) {
  forcedKernelRuntimeMode = mode;
}

export function getKernelRuntimeMode(): KernelRuntimeMode {
  return forcedKernelRuntimeMode;
}

export function getHostCapabilities(): HostCapabilities {
  const kind = detectHostKind();
  const localGenerationCapable = kind === "wails-desktop" && hasServiceMethod("Generate") && hasServiceMethod("Edit");
  const localPromptOptimizeCapable = kind === "wails-desktop" && hasServiceMethod("OptimizePrompt");
  const localModeEnabled = forcedKernelRuntimeMode !== "remote";
  return {
    localGeneration: localGenerationCapable && localModeEnabled,
    promptOptimization: localPromptOptimizeCapable && localModeEnabled,
    nativeFileDialogs: kind === "wails-desktop" || canInvokeAndroidMethod("OpenImageDialog"),
    nativeImageTransforms: kind === "wails-desktop" && hasServiceMethod("RotateImage") && hasServiceMethod("FlipImage") && hasServiceMethod("CropImage"),
    nativeHistoryFileIO: kind === "wails-desktop" || canInvokeAndroidMethod("ImportHistoryFromFile"),
    nativeOutputDirectoryPicker: hasServiceMethod("ChooseOutputDir") && kind !== "android-shell",
    secureCredentialStore: kind === "wails-desktop",
  };
}

export function EventsOn(eventName: string, callback: (...args: any[]) => void) {
  const offLocal = onLocalEvent(eventName, callback);
  const runtime = getRuntime();
  const offRuntime = runtime?.EventsOnMultiple
    ? runtime.EventsOnMultiple(eventName, callback, -1)
    : () => undefined;
  return () => {
    offLocal();
    offRuntime();
  };
}

export function EventsOff(eventName: string, ...additionalEventNames: string[]) {
  clearLocalEvents(eventName, ...additionalEventNames);
  getRuntime()?.EventsOff?.(eventName, ...additionalEventNames);
}

export function WindowSetSystemDefaultTheme() {
  getRuntime()?.WindowSetSystemDefaultTheme?.();
}

export function WindowSetLightTheme() {
  getRuntime()?.WindowSetLightTheme?.();
}

export function WindowSetDarkTheme() {
  getRuntime()?.WindowSetDarkTheme?.();
}

export function Generate(options: GenerateOptionsLike): Promise<JobStartedLike> {
  if (forcedKernelRuntimeMode === "local" && detectHostKind() !== "wails-desktop") {
    return Promise.reject(new Error("当前宿主不支持强制本地内核"));
  }
  if (getHostCapabilities().localGeneration) {
    return invokeService<JobStartedLike>("Generate", options);
  }
  return startRemoteJob({ ...options, mode: "generate" });
}

export function Edit(options: GenerateOptionsLike): Promise<JobStartedLike> {
  if (forcedKernelRuntimeMode === "local" && detectHostKind() !== "wails-desktop") {
    return Promise.reject(new Error("当前宿主不支持强制本地内核"));
  }
  if (getHostCapabilities().localGeneration) {
    return invokeService<JobStartedLike>("Edit", options);
  }
  return startRemoteJob({ ...options, mode: "edit" });
}

export function OptimizePrompt(options: PromptOptimizeOptionsLike): Promise<string> {
  if (forcedKernelRuntimeMode === "local" && detectHostKind() !== "wails-desktop") {
    return Promise.reject(new Error("当前宿主不支持强制本地内核"));
  }
  if (getHostCapabilities().promptOptimization) {
    return invokeService<string>("OptimizePrompt", options);
  }
  const controller = new AbortController();
  return optimizePromptRemote({
    apiKey: options.apiKey,
    prompt: options.prompt,
    mode: options.mode,
    baseURL: options.baseURL,
    textModelID: options.textModelID,
    imagePaths: options.imagePaths,
    imagePath: options.imagePath,
  }, controller.signal);
}

export function Cancel(jobId: string): Promise<void> {
  const remote = remoteJobControllers.get(jobId);
  if (remote) {
    remote.abort();
    remoteJobControllers.delete(jobId);
    return Promise.resolve();
  }
  if (hasServiceMethod("Cancel")) {
    return invokeService<void>("Cancel", jobId);
  }
  if (canInvokeAndroidMethod("Cancel")) {
    return invokeAndroid<void>("Cancel", jobId).catch(() => undefined);
  }
  return Promise.resolve();
}

export function OpenImageDialog(): Promise<SelectFileResponseLike> {
  if (hasServiceMethod("OpenImageDialog")) {
    return invokeService<SelectFileResponseLike>("OpenImageDialog").catch(() => openImageDialogFallback());
  }
  if (canInvokeAndroidMethod("OpenImageDialog")) {
    return invokeAndroid<SelectFileResponseLike>("OpenImageDialog").catch(() => openImageDialogFallback());
  }
  return openImageDialogFallback();
}

export function GetOutputDir(): Promise<string> {
  if (hasServiceMethod("GetOutputDir")) {
    return invokeService<string>("GetOutputDir");
  }
  if (canInvokeAndroidMethod("GetOutputDir")) {
    return invokeAndroid<string>("GetOutputDir");
  }
  return Promise.resolve("");
}

export function DeleteStoredAPIKey(user: string): Promise<void> {
  if (hasServiceMethod("DeleteStoredAPIKey")) {
    return invokeService<void>("DeleteStoredAPIKey", user);
  }
  if (canInvokeAndroidMethod("DeleteStoredAPIKey")) {
    return invokeAndroid<void>("DeleteStoredAPIKey", user);
  }
  setBrowserStoredAPIKey(user, "");
  return Promise.resolve();
}

export function GetStoredAPIKey(user: string): Promise<string> {
  if (hasServiceMethod("GetStoredAPIKey")) {
    return invokeService<string>("GetStoredAPIKey", user);
  }
  if (canInvokeAndroidMethod("GetStoredAPIKey")) {
    return invokeAndroid<string>("GetStoredAPIKey", user);
  }
  return Promise.resolve(browserStoredAPIKey(user));
}

export function SetStoredAPIKey(user: string, value: string): Promise<void> {
  if (hasServiceMethod("SetStoredAPIKey")) {
    return invokeService<void>("SetStoredAPIKey", user, value);
  }
  if (canInvokeAndroidMethod("SetStoredAPIKey")) {
    return invokeAndroid<void>("SetStoredAPIKey", user, value);
  }
  setBrowserStoredAPIKey(user, value);
  return Promise.resolve();
}

export function SaveImageAs(imageB64: string, suggestedName: string): Promise<string> {
  if (hasServiceMethod("SaveImageAs")) {
    return invokeService<string>("SaveImageAs", imageB64, suggestedName);
  }
  const mimeType = suggestedName.toLowerCase().endsWith(".jpg") || suggestedName.toLowerCase().endsWith(".jpeg")
    ? "image/jpeg"
    : suggestedName.toLowerCase().endsWith(".webp")
      ? "image/webp"
      : "image/png";
  return Promise.resolve(saveByDownload(new Blob([Uint8Array.from(atob(imageB64), (ch) => ch.charCodeAt(0))], { type: mimeType }), suggestedName));
}

export function ImportImageFromB64(imageB64: string, suggestedName: string): Promise<ImportedImageLike> {
  if (hasServiceMethod("ImportImageFromB64")) {
    return invokeService<ImportedImageLike>("ImportImageFromB64", imageB64, suggestedName)
      .catch(() => registerVirtualImage({ imageB64, suggestedName }));
  }
  if (canInvokeAndroidMethod("ImportImageFromB64")) {
    return invokeAndroid<ImportedImageLike>("ImportImageFromB64", imageB64, suggestedName)
      .catch(() => registerVirtualImage({ imageB64, suggestedName }));
  }
  return Promise.resolve(registerVirtualImage({ imageB64, suggestedName }));
}

export function RotateImage(path: string, degrees: number): Promise<ImportedImageLike> {
  if (isVirtualPath(path)) {
    return rotateVirtualImage(path, degrees);
  }
  return invokeService<ImportedImageLike>("RotateImage", path, degrees)
    .catch(async () => {
      const imported = await materializeReadablePathAsVirtual(path);
      return rotateVirtualImage(imported.path, degrees);
    });
}

export function FlipImage(path: string, horizontal: boolean): Promise<ImportedImageLike> {
  if (isVirtualPath(path)) {
    return flipVirtualImage(path, horizontal);
  }
  return invokeService<ImportedImageLike>("FlipImage", path, horizontal)
    .catch(async () => {
      const imported = await materializeReadablePathAsVirtual(path);
      return flipVirtualImage(imported.path, horizontal);
    });
}

export function CropImage(path: string, x: number, y: number, width: number, height: number): Promise<ImportedImageLike> {
  if (isVirtualPath(path)) {
    return cropVirtualImage(path, x, y, width, height);
  }
  return invokeService<ImportedImageLike>("CropImage", path, x, y, width, height)
    .catch(async () => {
      const imported = await materializeReadablePathAsVirtual(path);
      return cropVirtualImage(imported.path, x, y, width, height);
    });
}

export function ReadImageAsBase64(path: string): Promise<string> {
  if (isVirtualPath(path)) {
    return Promise.resolve(readVirtualImageAsBase64(path));
  }
  if (canInvokeAndroidMethod("ReadImageAsBase64")) {
    return invokeAndroid<string>("ReadImageAsBase64", path);
  }
  return invokeService<string>("ReadImageAsBase64", path);
}

export function ExportHistoryToFile(jsonContent: string): Promise<string> {
  if (hasServiceMethod("ExportHistoryToFile")) {
    return invokeService<string>("ExportHistoryToFile", jsonContent);
  }
  return Promise.resolve(saveByDownload(new Blob([jsonContent], { type: "application/json" }), `image-studio-history-${Date.now()}.json`));
}

export function ImportHistoryFromFile(): Promise<string> {
  if (hasServiceMethod("ImportHistoryFromFile")) {
    return invokeService<string>("ImportHistoryFromFile");
  }
  if (canInvokeAndroidMethod("ImportHistoryFromFile")) {
    return invokeAndroid<string>("ImportHistoryFromFile");
  }
  return importHistoryFallback();
}

export function RegisterTrustedOutputDir(root: string): Promise<void> {
  if (hasServiceMethod("RegisterTrustedOutputDir")) {
    return invokeService<void>("RegisterTrustedOutputDir", root);
  }
  return Promise.resolve();
}

export function SetOutputDir(path: string): Promise<void> {
  if (hasServiceMethod("SetOutputDir")) {
    return invokeService<void>("SetOutputDir", path);
  }
  if (canInvokeAndroidMethod("SetOutputDir")) {
    return invokeAndroid<void>("SetOutputDir", path);
  }
  return Promise.resolve();
}

export function ChooseOutputDir(): Promise<string> {
  if (hasServiceMethod("ChooseOutputDir")) {
    return invokeService<string>("ChooseOutputDir");
  }
  if (canInvokeAndroidMethod("ChooseOutputDir")) {
    return invokeAndroid<string>("ChooseOutputDir");
  }
  return GetOutputDir();
}

export function OpenOutputDir(): Promise<void> {
  if (hasServiceMethod("OpenOutputDir")) {
    return invokeService<void>("OpenOutputDir");
  }
  if (canInvokeAndroidMethod("OpenOutputDir")) {
    return invokeAndroid<void>("OpenOutputDir");
  }
  return Promise.reject(new Error(unsupportedMessage("OpenOutputDir")));
}

export function OpenExternalURL(url: string): Promise<void> {
  if (canInvokeAndroidMethod("OpenExternalURL")) {
    return invokeAndroid<void>("OpenExternalURL", url).catch(() => {
      const opened = typeof window !== "undefined" ? window.open(url, "_blank", "noopener,noreferrer") : null;
      if (!opened && typeof window !== "undefined") window.location.href = url;
    });
  }
  if (!hasServiceMethod("OpenExternalURL")) {
    if (typeof window !== "undefined") {
      const opened = window.open(url, "_blank", "noopener,noreferrer");
      if (!opened) window.location.href = url;
      return Promise.resolve();
    }
    return Promise.reject(new Error(unsupportedMessage("OpenExternalURL")));
  }
  return invokeService<void>("OpenExternalURL", url);
}

export function OpenFile(path: string): Promise<void> {
  if (isVirtualPath(path)) {
    return openVirtualPath(path);
  }
  if (canInvokeAndroidMethod("OpenFile")) {
    return invokeAndroid<void>("OpenFile", path);
  }
  return invokeService<void>("OpenFile", path);
}

export function ReadTextFile(path: string): Promise<string> {
  if (isVirtualPath(path)) {
    return Promise.resolve(readVirtualText(path));
  }
  if (canInvokeAndroidMethod("ReadTextFile")) {
    return invokeAndroid<string>("ReadTextFile", path);
  }
  return invokeService<string>("ReadTextFile", path);
}

export function probeCurrentUpstream(baseURL: string, apiKey: string, signal?: AbortSignal): Promise<void> {
  return probeUpstreamConnection(baseURL, apiKey, signal);
}

export function registerEphemeralLog(text: string, suggestedName = "raw-response.txt"): string {
  return registerVirtualText(text, suggestedName);
}
