import assert from "node:assert/strict";
import test from "node:test";

import {
  DEFAULT_PARTIAL_IMAGES,
  DEFAULT_REASONING_EFFORT,
  buildResponsesPayload,
  describeProblem,
  normalizePartialImages,
} from "../../../shared/kernel/requestModel.js";

test("Responses payload defaults partial_images to streaming preview count", () => {
  const payload = buildResponsesPayload({
    prompt: "cat",
    size: "1024x1024",
    quality: "low",
    outputFormat: "png",
    imageModelID: "gpt-image-2",
    textModelID: "gpt-5.5",
    requestPolicy: "openai",
  }, []);
  assert.equal(payload.tools[0].partial_images, DEFAULT_PARTIAL_IMAGES);
  assert.equal(payload.reasoning.effort, DEFAULT_REASONING_EFFORT);
});

test("normalizePartialImages clamps OpenAI range", () => {
  assert.equal(normalizePartialImages(0), 0);
  assert.equal(normalizePartialImages(-1), DEFAULT_PARTIAL_IMAGES);
  assert.equal(normalizePartialImages(2.8), 2);
  assert.equal(normalizePartialImages(9), 3);
});

test("Responses payload uses configured reasoning effort", () => {
  const payload = buildResponsesPayload({
    prompt: "cat",
    imageModelID: "gpt-image-2",
    textModelID: "gpt-5.5",
    reasoningEffort: "high",
  }, []);
  assert.equal(payload.reasoning.effort, "high");
});

test("describeProblem extracts refusal text from Responses SSE message events", () => {
  const raw = [
    'data: {"type":"response.output_item.done","item":{"type":"message","status":"completed","content":[{"type":"output_text","text":"抱歉，这个请求包含成人裸露，我无法生成这类真实照片风格图片。"}]}}',
    'data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"image_generation_call","status":"failed"}]}}',
  ].join("\n");
  assert.equal(describeProblem(raw), "抱歉，这个请求包含成人裸露，我无法生成这类真实照片风格图片。");
});
