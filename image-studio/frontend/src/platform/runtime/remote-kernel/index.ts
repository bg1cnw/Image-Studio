import { buildPromptOptimizePayload } from "../../../../../../shared/kernel/requestModel.js";
import {
  extractResponseErrorMessage,
  extractResponseText,
  fileNameFromPath,
  isRetryableRaw,
  isTransportishError,
  normalizeAPIMode,
  normalizeBaseURL,
  readRegisteredText,
  shouldUseAndroidNativeHTTP,
  sleepWithSignal,
  sourceToDataURL,
} from "./common.ts";
import { nativeHttpRequestText } from "./nativeHttp.ts";
import { requestImagesOnce } from "./images.ts";
import { requestResponsesOnce } from "./responses.ts";
import {
  MAX_ATTEMPTS,
  RETRY_BACKOFF_MS,
  RemoteKernelError,
  type RemotePromptOptimizeInput,
  type RemoteJobCallbacks,
  type RemoteJobRequest,
  type RemoteJobResult,
} from "./types.ts";

export * from "./types.ts";

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
          retryableRaw = isRetryableRaw(readRegisteredText(typed.rawPath));
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

export {
  MAX_ATTEMPTS,
  RETRY_BACKOFF_MS,
};
