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
const realAtob = globalThis.atob;
const realBtoa = globalThis.btoa;

function installBase64() {
  globalThis.atob = (value) => Buffer.from(value, "base64").toString("binary");
  globalThis.btoa = (value) => Buffer.from(value, "binary").toString("base64");
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

function installEnvironment() {
  globalThis.document = {
    body: { appendChild() {} },
    createElement(tag) {
      if (tag === "a") {
        return { href: "", download: "", click() {}, remove() {} };
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
    location: { href: "" },
    open() {
      return { closed: false };
    },
  };
  Object.defineProperty(globalThis, "navigator", {
    configurable: true,
    value: {
      userAgent: "Mozilla/5.0 (Linux; Android 14; Pixel)",
      platform: "Linux armv8l",
      userAgentData: { platform: "Android" },
    },
  });
  globalThis.URL = {
    ...URL,
    createObjectURL: () => "blob:mock",
    revokeObjectURL: () => {},
  };
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
    installStorage();
    installEnvironment();
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
    globalThis.atob = realAtob;
    globalThis.btoa = realBtoa;
  }
}

function loadRuntimeHost() {
  return import(`../src/lib/runtimeHost.ts?runtime-host-test=${Date.now()}-${Math.random().toString(36).slice(2)}`);
}

test("runtimeHost remote mode emits job lifecycle events", async () => {
  await withPatchedGlobals(async () => {
    globalThis.fetch = async (url) => {
      if (String(url).endsWith("/v1/responses")) {
        return new Response(
          'data: {"type":"response.created"}\n' +
          'data: {"type":"response.output_item.done","item":{"type":"image_generation_call","result":"YWJj","revised_prompt":"rev"}}\n',
          { status: 200, headers: { "content-type": "text/event-stream" } },
        );
      }
      throw new Error(`unexpected fetch ${url}`);
    };
  }, async () => {
    const runtimeHost = await loadRuntimeHost();
    runtimeHost.setKernelRuntimeMode("remote");

    const seen = { progress: [], result: [], error: [] };
    const started = await runtimeHost.Generate({
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
      transport: "auto",
      apiMode: "responses",
      noPromptRevision: false,
      concurrencyLimit: 0,
    });

    const offProgress = runtimeHost.EventsOn(`progress:${started.jobId}`, (payload) => {
      seen.progress.push(payload);
    });
    const offResult = runtimeHost.EventsOn(`result:${started.jobId}`, (payload) => {
      seen.result.push(payload);
    });
    const offError = runtimeHost.EventsOn(`error:${started.jobId}`, (payload) => {
      seen.error.push(payload);
    });

    await new Promise((resolve) => setImmediate(resolve));
    offProgress();
    offResult();
    offError();

    assert.equal(seen.error.length, 0);
    assert.ok(seen.result.length >= 1);
    assert.equal(seen.result[0].imageB64, "YWJj");
    assert.equal(seen.result[0].revisedPrompt, "rev");
  });
});

test("runtimeHost remote cancel aborts pending remote jobs", async () => {
  await withPatchedGlobals(async () => {
    globalThis.fetch = async (_url, init) => {
      await new Promise((_resolve, reject) => {
        init.signal.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true });
      });
      return new Response("", { status: 499 });
    };
  }, async () => {
    const runtimeHost = await loadRuntimeHost();
    runtimeHost.setKernelRuntimeMode("remote");

    const started = await runtimeHost.Generate({
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
      transport: "auto",
      apiMode: "responses",
      noPromptRevision: false,
      concurrencyLimit: 0,
    });

    await runtimeHost.Cancel(started.jobId);
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(typeof started.jobId, "string");
  });
});

test("runtimeHost can use Android invoke host capabilities directly", async () => {
  await withPatchedGlobals(async () => {
    const state = {
      apiKey: "",
      outputDir: "/sdcard/ImageStudio",
      imported: { path: "/sdcard/imports/source.png", imageB64: "YWJj" },
      selected: { path: "/sdcard/imports/picked.png", size: 3, imageB64: "YWJj" },
    };
    globalThis.window.AndroidImageStudio = {
      invoke(requestId, method, payloadJson) {
        const args = JSON.parse(payloadJson);
        queueMicrotask(() => {
          switch (method) {
            case "GetStoredAPIKey":
              window.__imageStudioNativeResolve?.(requestId, state.apiKey);
              break;
            case "SetStoredAPIKey":
              state.apiKey = args[1];
              window.__imageStudioNativeResolve?.(requestId, null);
              break;
            case "DeleteStoredAPIKey":
              state.apiKey = "";
              window.__imageStudioNativeResolve?.(requestId, null);
              break;
            case "GetOutputDir":
              window.__imageStudioNativeResolve?.(requestId, state.outputDir);
              break;
            case "SetOutputDir":
              state.outputDir = args[0];
              window.__imageStudioNativeResolve?.(requestId, null);
              break;
            case "ChooseOutputDir":
              window.__imageStudioNativeResolve?.(requestId, state.outputDir);
              break;
            case "OpenImageDialog":
              window.__imageStudioNativeResolve?.(requestId, state.selected);
              break;
            case "ImportImageFromB64":
              window.__imageStudioNativeResolve?.(requestId, state.imported);
              break;
            case "ReadImageAsBase64":
              window.__imageStudioNativeResolve?.(requestId, "YWJj");
              break;
            case "ImportHistoryFromFile":
              window.__imageStudioNativeResolve?.(requestId, '{"items":[]}');
              break;
            case "OpenFile":
            case "OpenOutputDir":
            case "OpenExternalURL":
              window.__imageStudioNativeResolve?.(requestId, null);
              break;
            default:
              window.__imageStudioNativeReject?.(requestId, `unsupported ${method}`);
          }
        });
      },
    };
  }, async () => {
    const runtimeHost = await loadRuntimeHost();
    runtimeHost.setKernelRuntimeMode("auto");

    assert.equal(runtimeHost.detectHostKind(), "android-shell");
    assert.equal(runtimeHost.getHostCapabilities().nativeFileDialogs, true);
    assert.equal(runtimeHost.getHostCapabilities().nativeHistoryFileIO, true);

    await runtimeHost.SetStoredAPIKey("profile:a", "sk-android");
    assert.equal(await runtimeHost.GetStoredAPIKey("profile:a"), "sk-android");
    await runtimeHost.DeleteStoredAPIKey("profile:a");
    assert.equal(await runtimeHost.GetStoredAPIKey("profile:a"), "");

    await runtimeHost.SetOutputDir("/sdcard/NewDir");
    assert.equal(await runtimeHost.GetOutputDir(), "/sdcard/NewDir");
    assert.equal(await runtimeHost.ChooseOutputDir(), "/sdcard/NewDir");

    const picked = await runtimeHost.OpenImageDialog();
    assert.equal(picked.path, "/sdcard/imports/picked.png");
    assert.equal(picked.imageB64, "YWJj");

    const imported = await runtimeHost.ImportImageFromB64("YWJj", "source.png");
    assert.equal(imported.path, "/sdcard/imports/source.png");
    assert.equal(await runtimeHost.ReadImageAsBase64(imported.path), "YWJj");
    assert.equal(await runtimeHost.ImportHistoryFromFile(), '{"items":[]}');
  });
});

test("runtimeHost Android invoke hooks coexist with shim-installed global callbacks", async () => {
  await withPatchedGlobals(async () => {
    const shimSeen = [];
    globalThis.window.__imageStudioNativeResolve = (requestId, payload) => {
      shimSeen.push({ kind: "resolve", requestId, payload });
    };
    globalThis.window.__imageStudioNativeReject = (requestId, message) => {
      shimSeen.push({ kind: "reject", requestId, message });
    };
    globalThis.window.AndroidImageStudio = {
      invoke(requestId, method, payloadJson) {
        const args = JSON.parse(payloadJson);
        queueMicrotask(() => {
          if (method === "GetStoredAPIKey") {
            window.__imageStudioNativeResolve?.(requestId, `echo:${args[0]}`);
            return;
          }
          if (method === "SetStoredAPIKey") {
            window.__imageStudioNativeResolve?.(requestId, null);
            window.__imageStudioNativeResolve?.("shim-owned-request", { method, payloadJson });
            return;
          }
          window.__imageStudioNativeResolve?.(requestId, null);
        });
      },
    };
    globalThis.__shimSeen = shimSeen;
  }, async () => {
    const runtimeHost = await loadRuntimeHost();
    const value = await runtimeHost.GetStoredAPIKey("profile:android");
    assert.equal(value, "echo:profile:android");
    await runtimeHost.SetStoredAPIKey("profile:android", "sk-value");
    assert.deepEqual(globalThis.__shimSeen, [
      {
        kind: "resolve",
        requestId: "shim-owned-request",
        payload: {
          method: "SetStoredAPIKey",
          payloadJson: JSON.stringify(["profile:android", "sk-value"]),
        },
      },
    ]);
  });
});
