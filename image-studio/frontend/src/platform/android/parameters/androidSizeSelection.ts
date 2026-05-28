import {
  buildSizeSelection,
  normalizeResolutionSelection,
  type AspectPreset,
  type ResolutionPreset,
} from "../../../components/panel/sizeCapabilities.ts";
import type { APIMode, RequestPolicy, SizeValue } from "../../../types/domain.ts";

type AndroidSizeSelectionInput = {
  apiMode: APIMode;
  requestPolicy: RequestPolicy;
  imageModelID?: string;
};

const DEFAULT_ASPECT_FROM_AUTO: Exclude<AspectPreset, "auto"> = "1:1";
const DEFAULT_RESOLUTION_FROM_AUTO: Exclude<ResolutionPreset, "auto"> = "1k";

export function buildAndroidAspectSizeSelection(
  aspect: AspectPreset,
  currentResolution: ResolutionPreset,
  input: AndroidSizeSelectionInput,
): SizeValue {
  if (aspect === "auto") return "auto";
  const normalizedResolution = normalizeResolutionSelection(currentResolution, input);
  return buildSizeSelection(
    aspect,
    normalizedResolution === "auto" ? DEFAULT_RESOLUTION_FROM_AUTO : normalizedResolution,
    input,
  );
}

export function buildAndroidResolutionSizeSelection(
  currentAspect: AspectPreset,
  resolution: ResolutionPreset,
  input: AndroidSizeSelectionInput,
): SizeValue {
  if (resolution === "auto") return "auto";
  return buildSizeSelection(
    currentAspect === "auto" ? DEFAULT_ASPECT_FROM_AUTO : currentAspect,
    normalizeResolutionSelection(resolution, input),
    input,
  );
}
