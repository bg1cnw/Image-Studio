import assert from "node:assert/strict";
import test from "node:test";

const realWindow = globalThis.window;
const realDocument = globalThis.document;
const realNavigator = globalThis.navigator;

function installPlatformEnv({ width, height, userAgent }) {
  globalThis.window = {
    innerWidth: width,
    innerHeight: height,
    addEventListener() {},
    removeEventListener() {},
    visualViewport: {
      addEventListener() {},
      removeEventListener() {},
    },
  };
  globalThis.document = {
    documentElement: {
      dataset: {},
    },
  };
  Object.defineProperty(globalThis, "navigator", {
    configurable: true,
    value: {
      userAgent,
      platform: "Linux armv8l",
      userAgentData: { platform: "Android" },
    },
  });
}

async function withPlatformEnv(env, run) {
  try {
    installPlatformEnv(env);
    return await run();
  } finally {
    globalThis.window = realWindow;
    globalThis.document = realDocument;
    Object.defineProperty(globalThis, "navigator", {
      configurable: true,
      value: realNavigator,
    });
  }
}

function loadPlatformModule() {
  return import(`../src/lib/platform.ts?platform-test=${Date.now()}-${Math.random().toString(36).slice(2)}`);
}

test("Android compact portrait stays on phone target", async () => {
  await withPlatformEnv({
    width: 412,
    height: 915,
    userAgent: "Mozilla/5.0 (Linux; Android 14; Pixel 8)",
  }, async () => {
    const platform = await loadPlatformModule();
    assert.equal(platform.targetPlatformForViewport(), "android");
    assert.equal(platform.readRuntimePlatformState().isAndroidPhone, true);
    assert.equal(platform.readRuntimePlatformState().isAndroidPad, false);
  });
});

test("Android medium width landscape upgrades to pad target", async () => {
  await withPlatformEnv({
    width: 700,
    height: 520,
    userAgent: "Mozilla/5.0 (Linux; Android 14; Tablet)",
  }, async () => {
    const platform = await loadPlatformModule();
    assert.equal(platform.targetPlatformForViewport(), "android-pad");
    assert.equal(platform.readRuntimePlatformState().isAndroidPad, true);
  });
});

test("Android medium width portrait remains phone target", async () => {
  await withPlatformEnv({
    width: 700,
    height: 1024,
    userAgent: "Mozilla/5.0 (Linux; Android 14; Tablet)",
  }, async () => {
    const platform = await loadPlatformModule();
    assert.equal(platform.targetPlatformForViewport(), "android");
    assert.equal(platform.readRuntimePlatformState().isAndroidPhone, true);
  });
});

test("Android expanded width portrait upgrades to pad target and applies attributes", async () => {
  await withPlatformEnv({
    width: 900,
    height: 1280,
    userAgent: "Mozilla/5.0 (Linux; Android 14; Foldable)",
  }, async () => {
    const platform = await loadPlatformModule();
    platform.applyPlatformAttributes(globalThis.document.documentElement);
    assert.equal(platform.targetPlatformForViewport(), "android-pad");
    assert.equal(globalThis.document.documentElement.dataset.platform, "android");
    assert.equal(globalThis.document.documentElement.dataset.targetPlatform, "android-pad");
    assert.equal(globalThis.document.documentElement.dataset.uiFamily, "android");
  });
});
