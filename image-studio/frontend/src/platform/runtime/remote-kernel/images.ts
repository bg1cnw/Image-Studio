import { buildImagesRequestBody } from "./requestPayloads.ts";
import {
  nowSeconds,
  registerRawText,
  resolveSourceDataURLs,
  shouldUseAndroidNativeHTTP,
} from "./common.ts";
import { nativeHttpRequestText } from "./nativeHttp.ts";
import {
  MAX_ATTEMPTS,
  RemoteKernelError,
  STATUS_INTERVAL_MS,
  type ExtractedImageResult,
  type RemoteJobCallbacks,
  type RemoteJobRequest,
  type RemoteJobResult,
} from "./types.ts";

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

export async function requestImagesOnce(
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
