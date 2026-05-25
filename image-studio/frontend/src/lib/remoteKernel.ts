import {
  detectImageMimeTypeFromBase64,
  imageExtensionForMimeType,
} from "./images.ts";
import {
  readVirtualText,
  registerVirtualText,
  sourceToDataURL,
} from "./virtualHostStore.ts";
import {
  hasAndroidInvokeBridge,
  invokeAndroidNative,
} from "./androidNativeInvoke.ts";

export type KernelImageSource = {
  path?: string;
  name?: string;
  mimeType?: string | null;
  imageB64?: string | null;
  imageBlob?: Blob | null;
};

export type RemoteGeneratePayload = {
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

type ProgressCallback = (stage: string, elapsedSeconds: number, bytesReceived: number) => void;

export type RemoteJobRequest = {
  payload: RemoteGeneratePayload;
  sourceImages?: KernelImageSource[];
};

export type RemoteJobCallbacks = {
  signal: AbortSignal;
  onLog?: (line: string) => void;
  onProgress?: ProgressCallback;
};

export type RemoteJobResult = {
  imageB64: string;
  revisedPrompt: string;
  sourceEvent: string;
  rawPath: string | null;
  prompt: string;
  mode: string;
};

export type RemotePromptOptimizeInput = {
  apiKey: string;
  prompt: string;
  mode: string;
  baseURL: string;
  textModelID: string;
  imagePaths?: string[];
  imagePath?: string;
  sourceImages?: KernelImageSource[];
};

const MAX_ATTEMPTS = 3;
const RETRY_BACKOFF_MS = 15_000;
const STATUS_INTERVAL_MS = 10_000;

export class RemoteKernelError extends Error {
  rawPath: string | null;

  constructor(message: string, rawPath: string | null = null) {
    super(message);
    this.name = "RemoteKernelError";
    this.rawPath = rawPath;
  }
}

type NativeTextResponse = {
  status: number;
  body: string;
  contentType?: string;
};

type ExtractedImageResult = {
  imageB64: string;
  revisedPrompt: string;
  sourceEvent: string;
};

function nowSeconds(startedAt: number): number {
  return Math.max(0, Math.floor((Date.now() - startedAt) / 1000));
}

function fileNameFromPath(path: string | undefined): string {
  if (!path) return "image.png";
  return path.split(/[\\/]/).pop() || "image.png";
}

async function resolveSourceDataURLs(
  sourceImages: KernelImageSource[] | undefined,
  payload: RemoteGeneratePayload,
): Promise<string[]> {
  const ordered = sourceImages?.length
    ? sourceImages
    : payload.imagePaths.map((path) => ({ path, name: fileNameFromPath(path) }));
  const out: string[] = [];
  for (const source of ordered) {
    const dataURL = await sourceToDataURL(source);
    if (dataURL) out.push(dataURL);
  }
  return out;
}

function normalizeBaseURL(raw: string): string {
  return raw.trim().replace(/\/+$/, "");
}

function normalizeAPIMode(apiMode: string): "responses" | "images" {
  return apiMode === "images" ? "images" : "responses";
}

function normalizeTextModel(modelID: string): string {
  return modelID.trim() || "gpt-5.5";
}

function normalizeImageModel(modelID: string): string {
  return modelID.trim() || "gpt-image-2";
}

function shouldUseAndroidNativeHTTP(): boolean {
  return typeof window !== "undefined" && hasAndroidInvokeBridge();
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  const chunkSize = 0x8000;
  for (let i = 0; i < bytes.length; i += chunkSize) {
    const chunk = bytes.subarray(i, i + chunkSize);
    binary += String.fromCharCode(...chunk);
  }
  return btoa(binary);
}

async function encodeRequestBody(
  body: BodyInit | null | undefined,
  headers?: Record<string, string>,
): Promise<{ bodyBase64: string; contentType: string }> {
  if (!body) {
    return { bodyBase64: "", contentType: headers?.["Content-Type"] || headers?.["content-type"] || "" };
  }
  if (typeof body === "string") {
    const bytes = new TextEncoder().encode(body);
    return {
      bodyBase64: bytesToBase64(bytes),
      contentType: headers?.["Content-Type"] || headers?.["content-type"] || "",
    };
  }
  const request = new Request("https://native-request.invalid", {
    method: "POST",
    headers,
    body,
  });
  const buffer = await request.arrayBuffer();
  return {
    bodyBase64: bytesToBase64(new Uint8Array(buffer)),
    contentType: request.headers.get("content-type") || headers?.["Content-Type"] || headers?.["content-type"] || "",
  };
}

async function nativeHttpRequestText(
  url: string,
  method: string,
  headers: Record<string, string>,
  body: BodyInit | null | undefined,
  signal?: AbortSignal,
): Promise<NativeTextResponse> {
  const requestKey = `native-http-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
  const encoded = await encodeRequestBody(body, headers);
  if (signal?.aborted) throw new DOMException("Aborted", "AbortError");
  let aborted = false;
  const onAbort = () => {
    aborted = true;
    void invokeAndroidNative<void>("CancelHttpRequest", requestKey).catch(() => undefined);
  };
  signal?.addEventListener("abort", onAbort, { once: true });
  try {
    const response = await invokeAndroidNative<NativeTextResponse>("HttpRequestText", {
      requestKey,
      url,
      method,
      headers,
      bodyBase64: encoded.bodyBase64,
      contentType: encoded.contentType,
    });
    if (aborted) throw new DOMException("Aborted", "AbortError");
    return response;
  } finally {
    signal?.removeEventListener("abort", onAbort);
  }
}

function buildResponsesPayload(
  payload: RemoteGeneratePayload,
  sourceDataURLs: string[],
): Record<string, unknown> {
  const size = payload.size || "1024x1024";
  const quality = payload.quality || "auto";
  const outputFormat = payload.outputFormat || "png";
  const content: Array<Record<string, unknown>> = [
    { type: "input_text", text: payload.prompt },
  ];
  for (const dataURL of sourceDataURLs) {
    content.push({ type: "input_image", image_url: dataURL });
  }
  const tool: Record<string, unknown> = {
    type: "image_generation",
    model: normalizeImageModel(payload.imageModelID),
    action: sourceDataURLs.length > 0 ? "edit" : "generate",
    size,
    quality,
    output_format: outputFormat,
    moderation: "low",
    partial_images: 0,
  };
  if (payload.maskB64) tool.mask = payload.maskB64;
  if (payload.seed) tool.seed = payload.seed;
  if (payload.negativePrompt.trim()) tool.negative_prompt = payload.negativePrompt.trim();

  const request: Record<string, unknown> = {
    model: normalizeTextModel(payload.textModelID),
    input: [{ role: "user", content }],
    tools: [tool],
    tool_choice: { type: "image_generation" },
    reasoning: { effort: "xhigh" },
    store: false,
    stream: true,
  };
  if (payload.noPromptRevision) {
    request.instructions = "You are a tool runner. Pass the user prompt to image_generation VERBATIM. DO NOT rewrite, expand, polish, or revise it in any way. Use the exact text the user gave.";
  }
  return request;
}

function buildPromptOptimizePayload(input: RemotePromptOptimizeInput, sourceDataURLs: string[]): Record<string, unknown> {
  let instruction = "Rewrite the user's image prompt into a clearer, more detailed prompt for image generation. Keep the meaning, preserve the requested subject, and only return the improved prompt text. Do not add explanations, labels, markdown, or quotes.";
  if (input.mode.trim() === "edit") {
    instruction += " Treat any attached images as reference context and preserve edit intent.";
  }
  const content: Array<Record<string, unknown>> = [
    { type: "input_text", text: `Original prompt:\n${input.prompt.trim()}` },
  ];
  for (const dataURL of sourceDataURLs) {
    content.push({ type: "input_image", image_url: dataURL });
  }
  return {
    model: normalizeTextModel(input.textModelID),
    instructions: instruction,
    input: [{ role: "user", content }],
    reasoning: { effort: "low" },
    store: false,
  };
}

function summarizeSSELine(line: string): string {
  const stripped = line.trim();
  if (!stripped) return "";
  if (stripped.startsWith(":")) return "收到接口保活信号";
  if (!stripped.startsWith("data: ")) return "";
  const payload = stripped.slice(6).trim();
  if (!payload || payload === "[DONE]") return "";
  let event: any;
  try {
    event = JSON.parse(payload);
  } catch {
    return "";
  }
  switch (event?.type) {
    case "response.created":
      return "请求已创建";
    case "response.in_progress":
      return "模型处理中";
    case "response.image_generation_call.in_progress":
      return "图片工具已启动";
    case "response.image_generation_call.generating":
      return "图片正在生成";
    case "response.image_generation_call.partial_image":
      return "已收到图片数据片段";
    case "response.output_item.done":
      if (event?.item?.type === "image_generation_call") {
        if (event.item.result) return "图片生成完成,正在保存";
        return `图片工具状态:${event.item.status || "未知"}`;
      }
      return "";
    case "response.completed":
      return "接口已完成";
    default:
      return event?.type ? `接口事件:${event.type}` : "";
  }
}

function extractImageResult(raw: string): ExtractedImageResult | null {
  let partialB64 = "";
  let partialPrompt = "";
  for (const line of raw.split(/\r?\n/)) {
    if (!line.startsWith("data: ")) continue;
    const payload = line.slice(6).trim();
    if (!payload || payload === "[DONE]") continue;
    let event: any;
    try {
      event = JSON.parse(payload);
    } catch {
      continue;
    }
    if (event?.type === "response.image_generation_call.partial_image" && event.partial_image_b64) {
      partialB64 = event.partial_image_b64;
      partialPrompt = event.revised_prompt || partialPrompt;
      continue;
    }
    if (event?.type === "response.output_item.done" && event?.item?.type === "image_generation_call") {
      if (event.item.result) {
        return {
          imageB64: event.item.result,
          revisedPrompt: event.item.revised_prompt || "",
          sourceEvent: "final",
        };
      }
      if (partialB64) {
        return {
          imageB64: partialB64,
          revisedPrompt: partialPrompt,
          sourceEvent: "partial",
        };
      }
    }
  }

  try {
    const parsed = JSON.parse(raw);
    const found = walkForImageCall(parsed);
    if (found?.result) {
      return {
        imageB64: found.result,
        revisedPrompt: found.revised_prompt || "",
        sourceEvent: "json",
      };
    }
  } catch {
    // ignore
  }

  if (partialB64) {
    return {
      imageB64: partialB64,
      revisedPrompt: partialPrompt,
      sourceEvent: "partial",
    };
  }

  return null;
}

function walkForImageCall(value: any): any | null {
  if (!value) return null;
  if (Array.isArray(value)) {
    for (const child of value) {
      const found = walkForImageCall(child);
      if (found) return found;
    }
    return null;
  }
  if (typeof value === "object") {
    if (value.type === "image_generation_call" && value.result) return value;
    for (const child of Object.values(value)) {
      const found = walkForImageCall(child);
      if (found) return found;
    }
  }
  return null;
}

const retryableMarkers = [
  "error code 524",
  "524: a timeout occurred",
  "error code 504",
  "gateway time-out",
  "service temporarily unavailable",
  "origin_gateway_timeout",
];

function isRetryableRaw(raw: string): boolean {
  const text = raw.trim();
  const lower = text.toLowerCase();
  if (retryableMarkers.some((marker) => lower.includes(marker))) return true;
  try {
    const data = JSON.parse(text);
    if (data?.retryable === true) return true;
    if ([502, 503, 504, 524].includes(Number(data?.status))) return true;
    const err = data?.error;
    if (err && typeof err === "object") {
      const message = String((err as any).message || "").toLowerCase();
      const type = String((err as any).type || "").toLowerCase();
      if (message.includes("temporarily unavailable")) return true;
      if (type === "api_error" || type === "server_error") return true;
    }
  } catch {
    // ignore
  }
  return false;
}

function describeAPIError(error: Record<string, unknown>): string {
  const code = String(error.code || "");
  const message = String(error.message || "");
  const type = String(error.type || "");

  switch (code.toLowerCase()) {
    case "moderation_blocked":
      return "🚫 上游内容审核拦截 · 生成被拒";
    case "content_policy_violation":
      return "🚫 上游内容政策拦截 (content_policy_violation)";
    case "rate_limit_exceeded":
      return `⏱ 上游限速 (rate_limit_exceeded)\n\n${message}`;
    case "insufficient_quota":
    case "billing_hard_limit_reached":
      return `💳 上游账户额度不足\n\n${message}`;
    case "model_not_found":
      return `🤷 上游找不到指定模型\n\n${message}`;
  }

  const parts = [];
  if (message) parts.push(message);
  const tail = [];
  if (code) tail.push(`code: ${code}`);
  if (type) tail.push(`type: ${type}`);
  if (tail.length > 0) parts.push(`(${tail.join(", ")})`);
  return parts.length > 0 ? `接口返回错误:${parts.join(" ")}` : "接口返回错误";
}

function describeProblem(raw: string): string {
  const text = raw.trim();
  if (!text) return "接口返回为空。";
  const lower = text.toLowerCase();
  if (lower.includes("error code 524") || lower.includes("524: a timeout occurred")) {
    return "Cloudflare 524:源站在超时时间内没有返回有效响应。";
  }
  if (lower.includes("error code 504") || lower.includes("gateway time-out")) {
    return "Cloudflare 504:源站网关超时。";
  }

  try {
    const data = JSON.parse(text);
    if (data?.error && typeof data.error === "object") {
      return describeAPIError(data.error);
    }
    if (typeof data?.message === "string" && data.message.trim()) {
      return `接口返回消息:${data.message.trim()}`;
    }
    if (data?.status && [502, 503, 504, 524].includes(Number(data.status))) {
      return `接口返回 ${data.status}:上游服务超时。`;
    }
  } catch {
    // ignore
  }

  for (const line of raw.split(/\r?\n/)) {
    if (!line.startsWith("data: ")) continue;
    const payload = line.slice(6).trim();
    if (!payload || payload === "[DONE]") continue;
    try {
      const event = JSON.parse(payload);
      if (event?.error && typeof event.error === "object") {
        return describeAPIError(event.error);
      }
      if (event?.response?.error && typeof event.response.error === "object") {
        return describeAPIError(event.response.error);
      }
    } catch {
      // ignore
    }
  }
  return "接口已返回内容,但没有发现 image_generation_call.result。";
}

function isTransportishError(error: unknown): boolean {
  const message = String((error as any)?.message || error || "").toLowerCase();
  return [
    "timeout",
    "networkerror",
    "network error",
    "failed to fetch",
    "load failed",
    "i/o timeout",
    "connection reset",
    "econnreset",
    "econnrefused",
    "gateway",
  ].some((marker) => message.includes(marker));
}

async function sleepWithSignal(signal: AbortSignal, ms: number): Promise<void> {
  if (signal.aborted) throw new DOMException("Aborted", "AbortError");
  await new Promise<void>((resolve, reject) => {
    const timer = globalThis.setTimeout(() => {
      cleanup();
      resolve();
    }, ms);
    const onAbort = () => {
      cleanup();
      reject(new DOMException("Aborted", "AbortError"));
    };
    const cleanup = () => {
      globalThis.clearTimeout(timer);
      signal.removeEventListener("abort", onAbort);
    };
    signal.addEventListener("abort", onAbort, { once: true });
  });
}

function registerRawText(kind: "responses" | "images" | "optimize", attempt: number, raw: string): string | null {
  if (!raw.trim()) return null;
  const ext = kind === "responses" ? "txt" : "json";
  return registerVirtualText(raw, `${kind}-response-attempt${attempt}.${ext}`);
}

async function requestResponsesOnce(
  request: RemoteJobRequest,
  attempt: number,
  callbacks: RemoteJobCallbacks,
): Promise<RemoteJobResult> {
  const sourceDataURLs = await resolveSourceDataURLs(request.sourceImages, request.payload);
  const body = JSON.stringify(buildResponsesPayload(request.payload, sourceDataURLs));
  const url = `${normalizeBaseURL(request.payload.baseURL)}/v1/responses`;
  const startedAt = Date.now();
  let lastStage = "等待接口响应";
  let bytesReceived = 0;
  let raw = "";
  callbacks.onLog?.(`第 ${attempt}/${MAX_ATTEMPTS} 次请求...`);
  callbacks.onProgress?.(lastStage, 0, 0);
  const ticker = globalThis.setInterval(() => {
    callbacks.onProgress?.(lastStage, nowSeconds(startedAt), bytesReceived);
  }, STATUS_INTERVAL_MS);
  try {
    if (shouldUseAndroidNativeHTTP()) {
      const response = await nativeHttpRequestText(
        url,
        "POST",
        {
          Authorization: `Bearer ${request.payload.apiKey}`,
          "Content-Type": "application/json",
          Accept: "text/event-stream, application/json",
        },
        body,
        callbacks.signal,
      );
      raw = response.body || "";
      const lines = raw.split(/\r?\n/);
      for (const line of lines) {
        bytesReceived += line.length;
        const summary = summarizeSSELine(line);
        if (summary) {
          lastStage = summary;
          callbacks.onLog?.(summary);
          callbacks.onProgress?.(lastStage, nowSeconds(startedAt), bytesReceived);
        }
      }
      const rawPath = registerRawText("responses", attempt, raw);
      if (response.status < 200 || response.status >= 300) {
        throw new RemoteKernelError(describeProblem(raw), rawPath);
      }
      const result = extractImageResult(raw);
      if (!result) {
        throw new RemoteKernelError(describeProblem(raw), rawPath);
      }
      return { ...result, rawPath, prompt: request.payload.prompt, mode: request.payload.mode };
    }
    const response = await fetch(url, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${request.payload.apiKey}`,
        "Content-Type": "application/json",
        Accept: "text/event-stream, application/json",
      },
      body,
      signal: callbacks.signal,
    });
    if (!response.body) {
      raw = await response.text();
      const rawPath = registerRawText("responses", attempt, raw);
      if (!response.ok) {
        throw new RemoteKernelError(describeProblem(raw), rawPath);
      }
      const result = extractImageResult(raw);
      if (!result) throw new RemoteKernelError("上游没有返回可用图片", rawPath);
      return { ...result, rawPath, prompt: request.payload.prompt, mode: request.payload.mode };
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let pending = "";
    try {
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        bytesReceived += value.byteLength;
        const chunk = decoder.decode(value, { stream: true });
        raw += chunk;
        pending += chunk;
        let newline = pending.indexOf("\n");
        while (newline >= 0) {
          const line = pending.slice(0, newline).replace(/\r$/, "");
          pending = pending.slice(newline + 1);
          const summary = summarizeSSELine(line);
          if (summary) {
            lastStage = summary;
            callbacks.onLog?.(summary);
            callbacks.onProgress?.(lastStage, nowSeconds(startedAt), bytesReceived);
          }
          newline = pending.indexOf("\n");
        }
      }
      raw += decoder.decode();
      if (pending.trim()) {
        const summary = summarizeSSELine(pending);
        if (summary) {
          lastStage = summary;
          callbacks.onLog?.(summary);
        }
      }
    } catch (error) {
      const fallback = extractImageResult(raw);
      if (fallback?.imageB64) {
        const rawPath = registerRawText("responses", attempt, raw);
        return { ...fallback, rawPath, prompt: request.payload.prompt, mode: request.payload.mode };
      }
      const rawPath = registerRawText("responses", attempt, raw);
      if (error instanceof RemoteKernelError) throw error;
      throw new RemoteKernelError(String((error as any)?.message || error), rawPath);
    }

    const rawPath = registerRawText("responses", attempt, raw);
    if (!response.ok) {
      throw new RemoteKernelError(describeProblem(raw), rawPath);
    }
    const result = extractImageResult(raw);
    if (!result) {
      throw new RemoteKernelError(describeProblem(raw), rawPath);
    }
    return { ...result, rawPath, prompt: request.payload.prompt, mode: request.payload.mode };
  } finally {
    globalThis.clearInterval(ticker);
  }
}

async function buildImagesRequestBody(
  request: RemoteJobRequest,
  sourceDataURLs: string[],
): Promise<{ url: string; headers?: Record<string, string>; body: BodyInit }> {
  const baseURL = normalizeBaseURL(request.payload.baseURL);
  const mode = request.payload.mode === "edit" ? "edit" : "generate";
  const imageModel = normalizeImageModel(request.payload.imageModelID);
  const size = request.payload.size || "1024x1024";
  const quality = request.payload.quality || "auto";
  const outputFormat = request.payload.outputFormat || "png";

  if (mode === "edit") {
    if (sourceDataURLs.length === 0) {
      throw new RemoteKernelError("图生图模式需要至少一张源图(请先添加参考图)");
    }
    const form = new FormData();
    for (let i = 0; i < sourceDataURLs.length; i++) {
      const dataURL = sourceDataURLs[i];
      const payload = dataURL.slice(dataURL.indexOf(",") + 1);
      const mimeType = dataURL.slice(5, dataURL.indexOf(";")) || "image/png";
      const ext = imageExtensionForMimeType(mimeType);
      form.append(i === 0 ? "image" : "image[]", new Blob([Uint8Array.from(atob(payload), (ch) => ch.charCodeAt(0))], { type: mimeType }), `source-${i + 1}.${ext}`);
    }
    if (request.payload.maskB64) {
      const maskMime = detectImageMimeTypeFromBase64(request.payload.maskB64) || "image/png";
      form.append("mask", new Blob([Uint8Array.from(atob(request.payload.maskB64), (ch) => ch.charCodeAt(0))], { type: maskMime }), "mask.png");
    }
    form.append("prompt", request.payload.prompt);
    form.append("model", imageModel);
    form.append("n", "1");
    form.append("size", size);
    form.append("quality", quality);
    form.append("output_format", outputFormat);
    form.append("response_format", "b64_json");
    if (request.payload.seed) form.append("seed", String(request.payload.seed));
    if (request.payload.negativePrompt.trim()) form.append("negative_prompt", request.payload.negativePrompt.trim());
    return { url: `${baseURL}/v1/images/edits`, body: form };
  }

  return {
    url: `${baseURL}/v1/images/generations`,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      model: imageModel,
      prompt: request.payload.prompt,
      n: 1,
      size,
      quality,
      output_format: outputFormat,
      response_format: "b64_json",
      ...(request.payload.seed ? { seed: request.payload.seed } : {}),
      ...(request.payload.negativePrompt.trim() ? { negative_prompt: request.payload.negativePrompt.trim() } : {}),
    }),
  };
}

function parseImagesResponse(raw: string, status: number): ExtractedImageResult {
  let parsed: any;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    if (status >= 400) {
      throw new RemoteKernelError(`上游返回 HTTP ${status}: ${raw.slice(0, 400)}`);
    }
    throw new RemoteKernelError(`解析 Images API 响应失败:${(error as any)?.message || error}`);
  }
  if (status >= 400) {
    if (parsed?.error?.message) {
      throw new RemoteKernelError(`上游返回 ${status}:${parsed.error.message}`);
    }
    throw new RemoteKernelError(`上游返回 HTTP ${status}`);
  }
  if (parsed?.error?.message) {
    throw new RemoteKernelError(`上游返回错误:${parsed.error.message}`);
  }
  const first = Array.isArray(parsed?.data) ? parsed.data[0] : null;
  if (!first?.b64_json) {
    if (first?.url) {
      throw new RemoteKernelError("上游返回 URL 而非 b64_json(不支持 response_format),请联系中转站启用 b64_json");
    }
    throw new RemoteKernelError("上游没有返回可用图片");
  }
  return {
    imageB64: first.b64_json,
    revisedPrompt: first.revised_prompt || "",
    sourceEvent: "images_api",
  };
}

async function requestImagesOnce(
  request: RemoteJobRequest,
  attempt: number,
  callbacks: RemoteJobCallbacks,
): Promise<RemoteJobResult> {
  const sourceDataURLs = await resolveSourceDataURLs(request.sourceImages, request.payload);
  const built = await buildImagesRequestBody(request, sourceDataURLs);
  const startedAt = Date.now();
  callbacks.onLog?.(`[Images API] 第 ${attempt}/${MAX_ATTEMPTS} 次请求...`);
  callbacks.onProgress?.("等待 Images API 返回(无 SSE 保活)", 0, 0);
  const ticker = globalThis.setInterval(() => {
    callbacks.onProgress?.("等待 Images API 返回(无 SSE 保活)", nowSeconds(startedAt), 0);
  }, STATUS_INTERVAL_MS);
  try {
    if (shouldUseAndroidNativeHTTP()) {
      const response = await nativeHttpRequestText(
        built.url,
        "POST",
        {
          Authorization: `Bearer ${request.payload.apiKey}`,
          Accept: "application/json",
          ...(built.headers ?? {}),
        },
        built.body,
        callbacks.signal,
      );
      const rawPath = registerRawText("images", attempt, response.body);
      const result = parseImagesResponse(response.body, response.status);
      return { ...result, rawPath, prompt: request.payload.prompt, mode: request.payload.mode };
    }
    const response = await fetch(built.url, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${request.payload.apiKey}`,
        Accept: "application/json",
        ...(built.headers ?? {}),
      },
      body: built.body,
      signal: callbacks.signal,
    });
    const raw = await response.text();
    const rawPath = registerRawText("images", attempt, raw);
    const result = parseImagesResponse(raw, response.status);
    return { ...result, rawPath, prompt: request.payload.prompt, mode: request.payload.mode };
  } catch (error) {
    if (error instanceof RemoteKernelError) throw error;
    throw new RemoteKernelError(String((error as any)?.message || error));
  } finally {
    globalThis.clearInterval(ticker);
  }
}

export async function runRemoteImageJob(
  request: RemoteJobRequest,
  callbacks: RemoteJobCallbacks,
): Promise<RemoteJobResult> {
  let lastError: RemoteKernelError | null = null;
  for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt++) {
    try {
      const apiMode = normalizeAPIMode(request.payload.apiMode);
      if (apiMode === "images") {
        return await requestImagesOnce(request, attempt, callbacks);
      }
      return await requestResponsesOnce(request, attempt, callbacks);
    } catch (error) {
      if (callbacks.signal.aborted) throw error;
      const typed = error instanceof RemoteKernelError
        ? error
        : new RemoteKernelError(String((error as any)?.message || error));
      lastError = typed;
      let retryableRaw = false;
      if (typed.rawPath) {
        try {
          retryableRaw = isRetryableRaw(readVirtualText(typed.rawPath));
        } catch {
          retryableRaw = false;
        }
      }
      const retryable = retryableRaw || isTransportishError(typed);
      if (attempt < MAX_ATTEMPTS && retryable) {
        callbacks.onLog?.(typed.message);
        callbacks.onLog?.(`${Math.floor(RETRY_BACKOFF_MS / 1000)} 秒后自动重试...`);
        await sleepWithSignal(callbacks.signal, RETRY_BACKOFF_MS);
        continue;
      }
      throw typed;
    }
  }
  throw lastError ?? new RemoteKernelError("多次请求后仍未成功");
}

function extractResponseText(raw: string): string {
  try {
    const parsed: any = JSON.parse(raw);
    if (typeof parsed?.output_text === "string" && parsed.output_text.trim()) {
      return parsed.output_text.trim();
    }
    if (Array.isArray(parsed?.output)) {
      for (const output of parsed.output) {
        if (!Array.isArray(output?.content)) continue;
        for (const content of output.content) {
          if (content?.type === "output_text" && typeof content?.text === "string" && content.text.trim()) {
            return content.text.trim();
          }
        }
      }
    }
  } catch {
    // ignore
  }
  return "";
}

function extractResponseErrorMessage(raw: string): string {
  try {
    const parsed: any = JSON.parse(raw);
    if (typeof parsed?.error?.message === "string" && parsed.error.message.trim()) {
      return parsed.error.message.trim();
    }
    if (typeof parsed?.message === "string" && parsed.message.trim()) {
      return parsed.message.trim();
    }
  } catch {
    // ignore
  }
  return raw.trim();
}

export async function optimizePromptRemote(
  input: RemotePromptOptimizeInput,
  signal: AbortSignal,
): Promise<string> {
  const mergedSources = input.sourceImages?.length
    ? input.sourceImages
    : [
        ...(input.imagePaths ?? []).map((path) => ({ path, name: fileNameFromPath(path) })),
        ...(input.imagePath ? [{ path: input.imagePath, name: fileNameFromPath(input.imagePath) }] : []),
      ];
  const sourceDataURLs: string[] = [];
  for (const source of mergedSources) {
    const dataURL = await sourceToDataURL(source);
    if (dataURL) sourceDataURLs.push(dataURL);
  }
  const url = `${normalizeBaseURL(input.baseURL)}/v1/responses`;
  const headers = {
    Authorization: `Bearer ${input.apiKey}`,
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  const body = JSON.stringify(buildPromptOptimizePayload(input, sourceDataURLs));
  const response = shouldUseAndroidNativeHTTP()
    ? await nativeHttpRequestText(url, "POST", headers, body, signal)
    : {
        status: 0,
        body: "",
      };
  const raw = shouldUseAndroidNativeHTTP()
    ? response.body
    : await (async () => {
        const webResponse = await fetch(url, {
          method: "POST",
          headers,
          body,
          signal,
        });
        const text = await webResponse.text();
        response.status = webResponse.status;
        return text;
      })();
  if (response.status < 200 || response.status >= 300) {
    throw new RemoteKernelError(`上游返回 ${response.status}:${extractResponseErrorMessage(raw)}`);
  }
  const text = extractResponseText(raw);
  if (!text) {
    throw new RemoteKernelError("上游没有返回可用的优化结果");
  }
  return text;
}

export async function probeUpstreamConnection(
  baseURL: string,
  apiKey: string,
  signal?: AbortSignal,
): Promise<void> {
  if (shouldUseAndroidNativeHTTP()) {
    const response = await nativeHttpRequestText(
      `${normalizeBaseURL(baseURL)}/v1/models`,
      "GET",
      {
        Authorization: `Bearer ${apiKey.trim()}`,
      },
      null,
      signal,
    );
    if (response.status < 200 || response.status >= 300) {
      throw new RemoteKernelError(`${response.status}${response.body ? ` ${response.body.slice(0, 160)}` : ""}`);
    }
    return;
  }
  const response = await fetch(`${normalizeBaseURL(baseURL)}/v1/models`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${apiKey.trim()}`,
    },
    signal,
  });
  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new RemoteKernelError(`${response.status}${text ? ` ${text.slice(0, 160)}` : ""}`);
  }
}
