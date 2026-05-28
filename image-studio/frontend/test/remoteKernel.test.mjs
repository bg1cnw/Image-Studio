import assert from "node:assert/strict";
import test from "node:test";

const realFetch = globalThis.fetch;
const realSetTimeout = globalThis.setTimeout;
const realClearTimeout = globalThis.clearTimeout;
const realSetInterval = globalThis.setInterval;
const realClearInterval = globalThis.clearInterval;
const realLocalStorage = globalThis.localStorage;
const realDocument = globalThis.document;
const realWindow = globalThis.window;
const realURL = globalThis.URL;
const realCreateObjectURL = globalThis.URL?.createObjectURL;
const realRevokeObjectURL = globalThis.URL?.revokeObjectURL;
const realAtob = globalThis.atob;
const realBtoa = globalThis.btoa;

function installBase64() {
  globalThis.atob = (value) => Buffer.from(value, "base64").toString("binary");
  globalThis.btoa = (value) => Buffer.from(value, "binary").toString("base64");
}

function installURLStubs() {
  const fakeURL = {
    ...URL,
    createObjectURL: () => "blob:mock",
    revokeObjectURL: () => {},
  };
  globalThis.URL = fakeURL;
}

function installStorage() {
  const store = new Map();
  globalThis.localStorage = {
    getItem(key) {
      return store.has(key) ? store.get(key) : null;
    },
    setItem(key, value) {
      store.set(key, String(value));
    },
    removeItem(key) {
      store.delete(key);
    },
  };
}

function installDocument() {
  globalThis.document = {
    body: {
      appendChild() {},
    },
    createElement(tag) {
      if (tag === "a") {
        return {
          href: "",
          download: "",
          click() {},
          remove() {},
        };
      }
      if (tag === "input") {
        return {
          type: "",
          accept: "",
          style: {},
          files: [],
          addEventListener() {},
          click() {},
          remove() {},
        };
      }
      if (tag === "canvas") {
        return {
          width: 0,
          height: 0,
          getContext() {
            return {
              translate() {},
              rotate() {},
              drawImage() {},
              scale() {},
            };
          },
          toBlob(callback) {
            callback(new Blob(["canvas"], { type: "image/png" }));
          },
        };
      }
      return {};
    },
  };
  globalThis.window = {
    open() {
      return { closed: false };
    },
    location: { href: "" },
  };
}

function installImmediateTimers() {
  globalThis.setTimeout = (fn, _ms, ...args) => {
    queueMicrotask(() => fn(...args));
    return 0;
  };
  globalThis.clearTimeout = () => {};
  globalThis.setInterval = () => 0;
  globalThis.clearInterval = () => {};
}

async function withPatchedGlobals(setup, run) {
  try {
    installBase64();
    installURLStubs();
    installStorage();
    installDocument();
    installImmediateTimers();
    await setup();
    return await run();
  } finally {
    globalThis.fetch = realFetch;
    globalThis.setTimeout = realSetTimeout;
    globalThis.clearTimeout = realClearTimeout;
    globalThis.setInterval = realSetInterval;
    globalThis.clearInterval = realClearInterval;
    globalThis.localStorage = realLocalStorage;
    globalThis.document = realDocument;
    globalThis.window = realWindow;
    globalThis.URL = realURL;
    if (globalThis.URL && realCreateObjectURL) globalThis.URL.createObjectURL = realCreateObjectURL;
    if (globalThis.URL && realRevokeObjectURL) globalThis.URL.revokeObjectURL = realRevokeObjectURL;
    globalThis.atob = realAtob;
    globalThis.btoa = realBtoa;
  }
}

function loadRemoteKernel() {
  return import(`../src/platform/runtime/remoteKernel.ts?test=${Date.now()}-${Math.random().toString(36).slice(2)}`);
}

test("runRemoteImageJob retries retryable responses and returns parsed SSE image", async () => {
  let calls = 0;
  await withPatchedGlobals(async () => {
    globalThis.fetch = async () => {
      calls += 1;
      if (calls === 1) {
        return new Response("<html>Error code 524 | 524: A timeout occurred</html>", {
          status: 524,
          headers: { "content-type": "text/html" },
        });
      }
      return new Response(
        'data: {"type":"response.output_item.done","item":{"type":"image_generation_call","result":"YWJj","revised_prompt":"rev"}}\n',
        { status: 200, headers: { "content-type": "text/event-stream" } },
      );
    };
  }, async () => {
    const kernel = await loadRemoteKernel();
    const result = await kernel.runRemoteImageJob(
      {
        payload: {
          apiKey: "key",
          mode: "generate",
          prompt: "cat",
          size: "1024x1024",
          quality: "low",
          outputFormat: "png",
          imagePaths: [],
          imagePath: "",
          maskB64: "",
          seed: 0,
          negativePrompt: "",
          baseURL: "https://upstream.example",
          textModelID: "gpt-5.5",
          imageModelID: "gpt-image-2",
          apiMode: "responses",
          noPromptRevision: false,
        },
      },
      { signal: new AbortController().signal },
    );
    assert.equal(calls, 2);
    assert.equal(result.imageB64, "YWJj");
    assert.equal(result.revisedPrompt, "rev");
    assert.equal(result.sourceEvent, "final");
    assert.ok(result.rawPath?.startsWith("memory://text/"));
  });
});

test("runRemoteImageJob parses Images API JSON mode", async () => {
  let captured = null;
  await withPatchedGlobals(async () => {
    globalThis.fetch = async (url, init) => {
      captured = {
        url: String(url),
        contentType: init.headers["Content-Type"] || init.headers["content-type"] || null,
        body: JSON.parse(init.body),
      };
      return new Response('{"data":[{"b64_json":"img-data","revised_prompt":"img-rev"}]}', {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    };
  }, async () => {
    const kernel = await loadRemoteKernel();
    const result = await kernel.runRemoteImageJob(
      {
        payload: {
          apiKey: "key",
          mode: "generate",
          prompt: "bird",
          size: "1024x1024",
          quality: "medium",
          outputFormat: "png",
          imagePaths: [],
          imagePath: "",
          maskB64: "",
          seed: 0,
          negativePrompt: "",
          baseURL: "https://upstream.example",
          textModelID: "",
          imageModelID: "gpt-image-2",
          apiMode: "images",
          noPromptRevision: false,
        },
      },
      { signal: new AbortController().signal },
    );
    assert.equal(captured.url, "https://upstream.example/v1/images/generations");
    assert.equal(captured.body.prompt, "bird");
    assert.equal(result.imageB64, "img-data");
    assert.equal(result.revisedPrompt, "img-rev");
    assert.equal(result.sourceEvent, "images_api");
  });
});

test("runRemoteImageJob sends Responses API mask as input_image_mask data URL", async () => {
  let captured = null;
  await withPatchedGlobals(async () => {
    globalThis.fetch = async (_url, init) => {
      captured = JSON.parse(init.body);
      return new Response(
        'data: {"type":"response.output_item.done","item":{"type":"image_generation_call","result":"YWJj","revised_prompt":"rev"}}\n',
        { status: 200, headers: { "content-type": "text/event-stream" } },
      );
    };
  }, async () => {
    const kernel = await loadRemoteKernel();
    await kernel.runRemoteImageJob(
      {
        payload: {
          apiKey: "key",
          mode: "edit",
          prompt: "cat",
          size: "1024x1024",
          quality: "low",
          outputFormat: "png",
          imagePaths: [],
          imagePath: "",
          maskB64: "iVBORw0KGgp0ZXN0",
          seed: 0,
          negativePrompt: "",
          baseURL: "https://upstream.example",
          textModelID: "gpt-5.5",
          imageModelID: "gpt-image-2",
          apiMode: "responses",
          requestPolicy: "openai",
          noPromptRevision: false,
        },
        sourceImages: [
          { imageB64: "iVBORw0KGgpzb3VyY2U=", name: "source.png", mimeType: "image/png" },
        ],
      },
      { signal: new AbortController().signal },
    );
    assert.equal(captured.tools[0].input_image_mask.image_url, "data:image/png;base64,iVBORw0KGgp0ZXN0");
    assert.equal(captured.tools[0].action, "edit");
  });
});

test("runRemoteImageJob sends Images API edit mask with image MIME type", async () => {
  let captured = null;
  await withPatchedGlobals(async () => {
    globalThis.fetch = async (url, init) => {
      captured = {
        url: String(url),
        body: init.body,
      };
      return new Response('{"data":[{"b64_json":"img-data","revised_prompt":"img-rev"}]}', {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    };
  }, async () => {
    const kernel = await loadRemoteKernel();
    await kernel.runRemoteImageJob(
      {
        payload: {
          apiKey: "key",
          mode: "edit",
          prompt: "bird",
          size: "1024x1024",
          quality: "medium",
          outputFormat: "png",
          imagePaths: [],
          imagePath: "",
          maskB64: "iVBORw0KGgpmYWtl",
          seed: 0,
          negativePrompt: "",
          baseURL: "https://upstream.example",
          textModelID: "",
          imageModelID: "gpt-image-2",
          apiMode: "images",
          requestPolicy: "openai",
          noPromptRevision: false,
        },
        sourceImages: [
          { imageB64: "iVBORw0KGgpzb3VyY2U=", name: "source.png", mimeType: "image/png" },
        ],
      },
      { signal: new AbortController().signal },
    );
    assert.equal(captured.url, "https://upstream.example/v1/images/edits");
    assert.ok(captured.body instanceof FormData);
    const mask = captured.body.get("mask");
    assert.ok(mask instanceof Blob);
    assert.equal(mask.type, "image/png");
  });
});

test("runRemoteImageJob omits relay-only fields by default and includes them in compat mode", async () => {
  const capturedBodies = [];
  await withPatchedGlobals(async () => {
    globalThis.fetch = async (_url, init) => {
      capturedBodies.push(JSON.parse(init.body));
      return new Response(
        'data: {"type":"response.output_item.done","item":{"type":"image_generation_call","result":"YWJj","revised_prompt":"rev"}}\n',
        { status: 200, headers: { "content-type": "text/event-stream" } },
      );
    };
  }, async () => {
    const kernel = await loadRemoteKernel();
    await kernel.runRemoteImageJob(
      {
        payload: {
          apiKey: "key",
          mode: "generate",
          prompt: "cat",
          size: "1024x1024",
          quality: "low",
          outputFormat: "png",
          imagePaths: [],
          imagePath: "",
          maskB64: "",
          seed: 123,
          negativePrompt: "avoid blur",
          baseURL: "https://upstream.example",
          textModelID: "gpt-5.5",
          imageModelID: "gpt-image-2",
          apiMode: "responses",
          requestPolicy: "openai",
          noPromptRevision: false,
        },
      },
      { signal: new AbortController().signal },
    );
    await kernel.runRemoteImageJob(
      {
        payload: {
          apiKey: "key",
          mode: "generate",
          prompt: "cat",
          size: "1024x1024",
          quality: "low",
          outputFormat: "png",
          imagePaths: [],
          imagePath: "",
          maskB64: "",
          seed: 123,
          negativePrompt: "avoid blur",
          baseURL: "https://upstream.example",
          textModelID: "gpt-5.5",
          imageModelID: "gpt-image-2",
          apiMode: "responses",
          requestPolicy: "compat",
          noPromptRevision: false,
        },
      },
      { signal: new AbortController().signal },
    );
    assert.equal(capturedBodies[0].tools[0].seed, undefined);
    assert.equal(capturedBodies[0].tools[0].negative_prompt, undefined);
    assert.ok(capturedBodies[0].instructions.includes("VERBATIM"));
    assert.equal(capturedBodies[1].tools[0].seed, 123);
    assert.equal(capturedBodies[1].tools[0].negative_prompt, "avoid blur");
    assert.ok(capturedBodies[1].instructions.includes("VERBATIM"));
  });
});

test("optimizePromptRemote extracts output_text", async () => {
  await withPatchedGlobals(async () => {
    globalThis.fetch = async () => new Response('{"output_text":"optimized prompt"}', {
      status: 200,
      headers: { "content-type": "application/json" },
    });
  }, async () => {
    const kernel = await loadRemoteKernel();
    const text = await kernel.optimizePromptRemote({
      apiKey: "key",
      prompt: "cat",
      mode: "generate",
      baseURL: "https://upstream.example",
      textModelID: "gpt-5.5",
      imagePaths: [],
      imagePath: "",
    }, new AbortController().signal);
    assert.equal(text, "optimized prompt");
  });
});

test("probeUpstreamConnection rejects non-2xx with summarized message", async () => {
  await withPatchedGlobals(async () => {
    globalThis.fetch = async () => new Response("forbidden", { status: 403 });
  }, async () => {
    const kernel = await loadRemoteKernel();
    await assert.rejects(
      () => kernel.probeUpstreamConnection("https://upstream.example", "key"),
      /403 forbidden/,
    );
  });
});

test("Android shell remote kernel can use native HTTP bridge to bypass browser fetch", async () => {
  await withPatchedGlobals(async () => {
    globalThis.window.AndroidImageStudio = {
      invoke(requestId, method, payloadJson) {
        const args = JSON.parse(payloadJson);
        queueMicrotask(() => {
          if (method === "HttpRequestText") {
            const payload = args[0];
            if (payload.url.endsWith("/v1/models")) {
              window.__imageStudioNativeResolve?.(requestId, {
                status: 200,
                body: '{"data":[{"id":"gpt-5.5"}]}',
                contentType: "application/json",
              });
              return;
            }
            if (payload.url.endsWith("/v1/responses")) {
              window.__imageStudioNativeResolve?.(requestId, {
                status: 200,
                body: 'data: {"type":"response.output_item.done","item":{"type":"image_generation_call","result":"YW5kcm9pZA==","revised_prompt":"native bridge"}}\n',
                contentType: "text/event-stream",
              });
              return;
            }
          }
          if (method === "CancelHttpRequest") {
            window.__imageStudioNativeResolve?.(requestId, null);
            return;
          }
          window.__imageStudioNativeReject?.(requestId, `unsupported ${method}`);
        });
      },
    };
    Object.defineProperty(globalThis, "navigator", {
      configurable: true,
      value: {
        userAgent: "Mozilla/5.0 (Linux; Android 16; Pixel)",
        platform: "Linux armv8l",
        userAgentData: { platform: "Android" },
      },
    });
    globalThis.fetch = async () => {
      throw new Error("browser fetch should not be used in Android native HTTP mode");
    };
  }, async () => {
    const kernel = await loadRemoteKernel();
    await kernel.probeUpstreamConnection("https://upstream.example", "key");
    const result = await kernel.runRemoteImageJob(
      {
        payload: {
          apiKey: "key",
          mode: "generate",
          prompt: "cat",
          size: "1024x1024",
          quality: "low",
          outputFormat: "png",
          imagePaths: [],
          imagePath: "",
          maskB64: "",
          seed: 0,
          negativePrompt: "",
          baseURL: "https://upstream.example",
          textModelID: "gpt-5.5",
          imageModelID: "gpt-image-2",
          apiMode: "responses",
          noPromptRevision: false,
        },
      },
      { signal: new AbortController().signal },
    );
    assert.equal(result.imageB64, "YW5kcm9pZA==");
    assert.equal(result.revisedPrompt, "native bridge");
  });
});
