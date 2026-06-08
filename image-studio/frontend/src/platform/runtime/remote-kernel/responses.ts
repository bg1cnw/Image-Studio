import {
  describeProblem,
  normalizeBaseURL,
  nowSeconds,
  registerRawText,
  resolveSourceDataURLs,
  shouldUseAndroidNativeHTTP,
} from "./common.ts";
import { nativeHttpRequestText } from "./nativeHttp.ts";
import { buildResponsesPayload } from "./requestPayloads.ts";
import {
  MAX_ATTEMPTS,
  RemoteKernelError,
  STATUS_INTERVAL_MS,
  type ExtractedImageResult,
  type RemoteJobCallbacks,
  type RemoteJobRequest,
  type RemoteJobResult,
} from "./types.ts";

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

function summarizeSSEEvent(event: any): string {
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

function parseSSELineEvent(line: string): any | null {
  const stripped = line.trim();
  if (!stripped.startsWith("data: ")) return null;
  const payload = stripped.slice(6).trim();
  if (!payload || payload === "[DONE]") return null;
  try {
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

function parseNativeProgressPayload(payload: unknown): { line: string; event: any | null } {
  if (typeof payload === "string") {
    return { line: payload, event: parseSSELineEvent(payload) };
  }
  if (!payload || typeof payload !== "object") {
    return { line: "", event: null };
  }
  const line = typeof (payload as { line?: unknown }).line === "string"
    ? (payload as { line: string }).line
    : "";
  const structured = (payload as { event?: unknown }).event;
  const event = structured && typeof structured === "object"
    ? structured
    : parseSSELineEvent(line);
  return { line, event };
}

function emitPartialPreview(event: any, callbacks: RemoteJobCallbacks) {
  if (event?.type !== "response.image_generation_call.partial_image") return;
  if (!event.partial_image_b64) return;
  callbacks.onPartialImage?.({
    imageB64: event.partial_image_b64,
    revisedPrompt: event.revised_prompt || undefined,
    partialImageIndex: typeof event.partial_image_index === "number" ? event.partial_image_index : undefined,
    sourceEvent: "responses_partial",
  });
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

function extractImageResult(raw: string): ExtractedImageResult | null {
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

  return null;
}

export async function requestResponsesOnce(
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
    const proxyMode = request.payload.proxyMode === "none" || request.payload.proxyMode === "custom" ? request.payload.proxyMode : "system";
    if (shouldUseAndroidNativeHTTP()) {
      let receivedNativeStreamPayload = false;
      const consumeNativePayload = (payload: unknown) => {
        receivedNativeStreamPayload = true;
        const parsed = parseNativeProgressPayload(payload);
        if (parsed.line) {
          bytesReceived += parsed.line.length + 1;
        }
        emitPartialPreview(parsed.event, callbacks);
        const summary = parsed.event ? summarizeSSEEvent(parsed.event) : summarizeSSELine(parsed.line);
        if (summary) {
          lastStage = summary;
          callbacks.onLog?.(summary);
          callbacks.onProgress?.(lastStage, nowSeconds(startedAt), bytesReceived);
        }
      };
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
        consumeNativePayload,
        { proxyMode, proxyURL: request.payload.proxyURL || "" },
      );
      raw = response.body || "";
      if (response.resultImageB64) {
        return {
          imageB64: response.resultImageB64,
          revisedPrompt: response.revisedPrompt || "",
          sourceEvent: response.sourceEvent || "final",
          rawPath: response.rawPath || null,
          prompt: request.payload.prompt,
          mode: request.payload.mode,
        };
      }
      if (!receivedNativeStreamPayload) {
        for (const line of raw.split(/\r?\n/)) consumeNativePayload(line);
      }
      const rawPath = response.rawPath || registerRawText("responses", attempt, raw);
      if (response.status < 200 || response.status >= 300) {
        throw new RemoteKernelError(describeProblem(raw), rawPath);
      }
      const result = extractImageResult(raw);
      if (!result) {
        throw new RemoteKernelError(describeProblem(raw), rawPath);
      }
      return { ...result, rawPath, prompt: request.payload.prompt, mode: request.payload.mode };
    }
    if (proxyMode !== "system") {
      throw new RemoteKernelError("当前远程内核不能控制代理,请切回本地内核或使用 Android 原生运行");
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
          emitPartialPreview(parseSSELineEvent(line), callbacks);
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
        emitPartialPreview(parseSSELineEvent(pending), callbacks);
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
