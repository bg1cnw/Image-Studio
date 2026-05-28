import { invokeAndroidNative } from "../../android/nativeInvoke.ts";
import type { NativeTextResponse } from "./types.ts";

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

export async function nativeHttpRequestText(
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
