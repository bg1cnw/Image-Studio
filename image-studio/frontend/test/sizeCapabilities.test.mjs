import assert from "node:assert/strict";
import test from "node:test";

const caps = await import("../src/components/panel/sizeCapabilities.ts");

test("gpt-image paths expose explicit 2K/4K resolution presets", () => {
  const values = caps.availableResolutionPresets({
    apiMode: "responses",
    requestPolicy: "openai",
    imageModelID: "gpt-image-2",
  });
  assert.ok(values.includes("2k"));
  assert.ok(values.includes("4k"));
  assert.equal(
    caps.buildSizeSelection("16:9", "4k", {
      apiMode: "responses",
      requestPolicy: "openai",
      imageModelID: "gpt-image-2",
    }),
    "3840x2160",
  );
});

test("blank image model falls back to default gpt-image-2 capabilities", () => {
  const input = {
    apiMode: "responses",
    requestPolicy: "openai",
    imageModelID: "",
  };
  const values = caps.availableResolutionPresets(input);
  assert.ok(values.includes("2k"));
  assert.ok(values.includes("4k"));
  assert.deepEqual(
    caps.listAspectPresetOptions(input).map((item) => item.value),
    ["auto", "1:1", "3:2", "2:3", "16:9", "9:16"],
  );
  assert.equal(caps.buildSizeSelection("9:16", "4k", input), "2160x3840");
});

test("legacy gpt-image models stay on documented base sizes", () => {
  const input = {
    apiMode: "images",
    requestPolicy: "openai",
    imageModelID: "gpt-image-1.5",
  };
  assert.deepEqual(caps.availableResolutionPresets(input), ["auto", "1k"]);
  assert.equal(
    caps.buildSizeSelection("16:9", "1k", input),
    "1024x1024",
  );
  assert.equal(
    caps.buildSizeSelection("3:2", "1k", input),
    "1536x1024",
  );
});

test("dall-e-3 uses only official aspect/size combinations", () => {
  const input = {
    apiMode: "images",
    requestPolicy: "openai",
    imageModelID: "dall-e-3",
  };
  const values = caps.availableResolutionPresets(input);
  assert.deepEqual(values, ["1k"]);
  const aspects = caps.listAspectPresetOptions(input).map((item) => item.value);
  assert.deepEqual(aspects, ["1:1", "7:4", "4:7"]);
  assert.equal(caps.buildSizeSelection("7:4", "1k", input), "1792x1024");
  assert.equal(caps.buildSizeSelection("4:7", "1k", input), "1024x1792");
});

test("dall-e-2 stays on square sizes only", () => {
  const input = {
    apiMode: "images",
    requestPolicy: "openai",
    imageModelID: "dall-e-2",
  };
  assert.deepEqual(caps.availableResolutionPresets(input), ["256", "512", "1k"]);
  const aspects = caps.listAspectPresetOptions(input).map((item) => item.value);
  assert.deepEqual(aspects, ["1:1"]);
  assert.equal(caps.buildSizeSelection("1:1", "256", input), "256x256");
  assert.equal(caps.buildSizeSelection("3:2", "1k", input), "1024x1024");
});

test("non-gpt-image openai-standard paths stay on base resolution presets", () => {
  const values = caps.availableResolutionPresets({
    apiMode: "responses",
    requestPolicy: "openai",
    imageModelID: "custom-relay-image",
  });
  assert.ok(!values.includes("2k"));
  assert.ok(!values.includes("4k"));
  assert.equal(caps.normalizeSizeSelection("3840x2160", {
    apiMode: "responses",
    requestPolicy: "openai",
    imageModelID: "custom-relay-image",
  }), "1536x864");
});

test("compat mode can keep large resolution presets available for compatible relays", () => {
  const values = caps.availableResolutionPresets({
    apiMode: "responses",
    requestPolicy: "compat",
    imageModelID: "relay-image-model",
  });
  assert.ok(values.includes("2k"));
  assert.ok(values.includes("4k"));
});

test("ratio stays independent from resolution preset", () => {
  assert.equal(
    caps.buildSizeSelection("1:1", "2k", {
      apiMode: "responses",
      requestPolicy: "openai",
      imageModelID: "gpt-image-2",
    }),
    "2048x2048",
  );
  assert.equal(
    caps.buildSizeSelection("9:16", "4k", {
      apiMode: "responses",
      requestPolicy: "openai",
      imageModelID: "gpt-image-2",
    }),
    "2160x3840",
  );
});

test("explicit aspect selection can leave Auto size", () => {
  assert.equal(
    caps.buildAspectSizeSelection("9:16", "auto", {
      apiMode: "responses",
      requestPolicy: "openai",
      imageModelID: "gpt-image-2",
    }),
    "864x1536",
  );
});

test("explicit resolution selection can leave Auto size", () => {
  assert.equal(
    caps.buildResolutionSizeSelection("auto", "2k", {
      apiMode: "responses",
      requestPolicy: "openai",
      imageModelID: "gpt-image-2",
    }),
    "2048x2048",
  );
});

test("explicit Auto selections keep upstream-determined size", () => {
  const input = {
    apiMode: "responses",
    requestPolicy: "openai",
    imageModelID: "gpt-image-2",
  };
  assert.equal(caps.buildAspectSizeSelection("auto", "2k", input), "auto");
  assert.equal(caps.buildResolutionSizeSelection("16:9", "auto", input), "auto");
});

test("custom aspect ratios can build sizes and round-trip back to the active custom button", () => {
  const input = {
    apiMode: "responses",
    requestPolicy: "compat",
    imageModelID: "relay-image-model",
  };
  const customRatios = [
    { id: "4:5", label: "4:5", width: 4, height: 5, createdAt: 1 },
  ];
  const value = caps.buildCustomAspectValue("4:5");
  const size = caps.buildAspectSizeSelection(value, "2k", input, customRatios);
  assert.equal(size, "1496x1864");
  assert.equal(caps.deriveAspectPreset(size, customRatios), value);
  assert.equal(caps.deriveResolutionPreset(size), "2k");
});
