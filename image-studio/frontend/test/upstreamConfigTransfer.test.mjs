import assert from "node:assert/strict";
import test from "node:test";

test("buildUpstreamConfigExportFile includes profile metadata and api keys", async () => {
  const mod = await import(`../src/lib/upstreamConfigTransfer.ts?upstream-config-transfer=${Date.now()}-${Math.random().toString(36).slice(2)}`);
  const payload = mod.buildUpstreamConfigExportFile([
    {
      id: "p-1",
      name: "主配置",
      apiMode: "responses",
      requestPolicy: "openai",
      imagesNewAPICompat: false,
      baseURL: "https://relay.example.com",
      textModelID: "gpt-5.5",
      imageModelID: "gpt-image-2",
      reasoningEffort: "xhigh",
      concurrencyLimit: 3,
      createdAt: 1,
      lastUsedAt: 2,
    },
  ], "p-1", { "p-1": "sk-live" });

  assert.equal(payload.version, 1);
  assert.equal(payload.activeProfileId, "p-1");
  assert.equal(payload.profiles[0].apiKey, "sk-live");
});

test("parseUpstreamConfigImportFile normalizes profiles and preserves embedded api keys", async () => {
  const mod = await import(`../src/lib/upstreamConfigTransfer.ts?upstream-config-transfer=${Date.now()}-${Math.random().toString(36).slice(2)}`);
  const parsed = mod.parseUpstreamConfigImportFile(JSON.stringify({
    version: 1,
    activeProfileId: "p-2",
    profiles: [
      {
        id: "p-2",
        name: "备用",
        apiMode: "images",
        responsesTransport: "sse",
        requestPolicy: "compat",
        imagesNewAPICompat: true,
        baseURL: " https://img.example.com/ ",
        textModelID: "ignored",
        imageModelID: "gpt-image-2",
        reasoningEffort: "medium",
        concurrencyLimit: 5.9,
        fallbackProfileId: "backup-1",
        createdAt: 10,
        lastUsedAt: 11,
        apiKey: " sk-backup ",
      },
    ],
  }));

  assert.deepEqual(parsed, {
    activeProfileId: "p-2",
    profiles: [
      {
        id: "p-2",
        name: "备用",
        apiMode: "images",
        responsesTransport: "sse",
        requestPolicy: "compat",
        imagesNewAPICompat: true,
        baseURL: "https://img.example.com/",
        textModelID: "ignored",
        imageModelID: "gpt-image-2",
        reasoningEffort: "medium",
        concurrencyLimit: 5,
        fallbackProfileId: "backup-1",
        createdAt: 10,
        lastUsedAt: 11,
        apiKey: "sk-backup",
      },
    ],
  });
});

test("parseUpstreamConfigImportFile rejects empty profile payloads", async () => {
  const mod = await import(`../src/lib/upstreamConfigTransfer.ts?upstream-config-transfer=${Date.now()}-${Math.random().toString(36).slice(2)}`);
  assert.throws(
    () => mod.parseUpstreamConfigImportFile(JSON.stringify({ version: 1, profiles: [] })),
    /没有可导入的上游配置/,
  );
});
