package ui

import (
	"strings"

	"github.com/yuanhua/image-gptcodex/pkg/client"
)

type choice struct {
	Label string
	Value string
}

type aspectChoice struct {
	Value string
	Label string
	W     int
	H     int
	Auto  bool
}

var (
	modeChoices = []choice{
		{"文生图", string(client.ModeGenerate)},
		{"图生图", string(client.ModeEdit)},
	}
	apiChoices = []choice{
		{"Responses", string(client.APIModeResponses)},
		{"Images", string(client.APIModeImages)},
	}
	sizeChoices = []choice{
		{"自适应 auto", "auto"},
		{"方形 1024x1024", "1024x1024"},
		{"横版 1536x1024", "1536x1024"},
		{"竖版 1024x1536", "1024x1536"},
		{"横版 1536x864", "1536x864"},
		{"竖版 864x1536", "864x1536"},
		{"2K 方形 2048x2048", "2048x2048"},
		{"2K 横版 2048x1360", "2048x1360"},
		{"2K 竖版 1360x2048", "1360x2048"},
		{"2K 横版 2048x1152", "2048x1152"},
		{"2K 竖版 1152x2048", "1152x2048"},
		{"4K 方形 2880x2880", "2880x2880"},
		{"4K 横版 3456x2304", "3456x2304"},
		{"4K 竖版 2304x3456", "2304x3456"},
		{"4K 横版 3840x2160", "3840x2160"},
		{"4K 竖版 2160x3840", "2160x3840"},
	}
	aspectChoices = []aspectChoice{
		{Value: "auto", Label: "Auto", W: 18, H: 18, Auto: true},
		{Value: "1:1", Label: "1:1", W: 18, H: 18},
		{Value: "3:2", Label: "3:2", W: 22, H: 14},
		{Value: "2:3", Label: "2:3", W: 14, H: 20},
		{Value: "16:9", Label: "16:9", W: 24, H: 13},
		{Value: "9:16", Label: "9:16", W: 12, H: 22},
	}
	resolutionChoices = []choice{
		{"自动", "auto"},
		{"1K", "1k"},
		{"2K", "2k"},
		{"4K", "4k"},
	}
	qualityChoices = []choice{
		{"自动", "auto"},
		{"精修", "high"},
		{"标准", "medium"},
		{"快速", "low"},
	}
	formatChoices = []choice{
		{"PNG", "png"},
		{"JPEG", "jpeg"},
		{"WebP", "webp"},
	}
	policyChoices = []choice{
		{"OpenAI 标准", string(client.RequestPolicyOpenAI)},
		{"兼容中转扩展", string(client.RequestPolicyCompat)},
	}
	proxyChoices = []choice{
		{"系统配置", client.ProxyModeSystem},
		{"不使用", client.ProxyModeNone},
		{"自定义", client.ProxyModeCustom},
	}
	styleChoices = []choice{
		{"赛博朋克", "cyberpunk"},
		{"二次元", "anime"},
		{"插画", "illust"},
		{"3D 渲染", "3d"},
		{"国风", "chinese"},
	}
	batchCountChoices = []choice{
		{"1", "1"},
		{"2", "2"},
		{"4", "4"},
		{"6", "6"},
		{"8", "8"},
		{"9", "9"},
	}
	styleSuffixes = map[string]string{
		"cyberpunk": "cyberpunk style, neon lights, glowing reflections, futuristic",
		"anime":     "anime style, cel shading, vibrant colors, detailed illustration",
		"illust":    "modern illustration, flat colors, clean lines",
		"3d":        "3D render, octane render, ray tracing, glossy surfaces, studio lighting",
		"chinese":   "traditional Chinese painting style, ink wash, misty landscape",
	}
	sizeMatrix = map[string]map[string]string{
		"1:1": {
			"1k": "1024x1024",
			"2k": "2048x2048",
			"4k": "2880x2880",
		},
		"3:2": {
			"1k": "1536x1024",
			"2k": "2048x1360",
			"4k": "3456x2304",
		},
		"2:3": {
			"1k": "1024x1536",
			"2k": "1360x2048",
			"4k": "2304x3456",
		},
		"16:9": {
			"1k": "1536x864",
			"2k": "2048x1152",
			"4k": "3840x2160",
		},
		"9:16": {
			"1k": "864x1536",
			"2k": "1152x2048",
			"4k": "2160x3840",
		},
	}
	sizeToAspect = map[string]string{
		"auto":      "auto",
		"1024x1024": "1:1",
		"2048x2048": "1:1",
		"2880x2880": "1:1",
		"1536x1024": "3:2",
		"2048x1360": "3:2",
		"3456x2304": "3:2",
		"1024x1536": "2:3",
		"1360x2048": "2:3",
		"2304x3456": "2:3",
		"1536x864":  "16:9",
		"2048x1152": "16:9",
		"3840x2160": "16:9",
		"864x1536":  "9:16",
		"1152x2048": "9:16",
		"2160x3840": "9:16",
	}
	sizeToResolution = map[string]string{
		"auto":      "auto",
		"1024x1024": "1k",
		"1536x1024": "1k",
		"1024x1536": "1k",
		"1536x864":  "1k",
		"864x1536":  "1k",
		"2048x2048": "2k",
		"2048x1360": "2k",
		"1360x2048": "2k",
		"2048x1152": "2k",
		"1152x2048": "2k",
		"2880x2880": "4k",
		"3456x2304": "4k",
		"2304x3456": "4k",
		"3840x2160": "4k",
		"2160x3840": "4k",
	}
)

func (a *App) modeLabel() string {
	if a.mode == string(client.ModeEdit) {
		return "图生图"
	}
	return "文生图"
}

func sizeChoiceLabel(value string) string {
	return choiceLabel(sizeChoices, value)
}

func qualityChoiceLabel(value string) string {
	return choiceLabel(qualityChoices, value)
}

func styleChoiceLabel(value string) string {
	return choiceLabel(styleChoices, value)
}

func chooseStyleSummary(value string) string {
	if value == "" {
		return "默认风格"
	}
	return styleChoiceLabel(value)
}

func deriveAspectPreset(size string) string {
	if value, ok := sizeToAspect[size]; ok {
		return value
	}
	return "1:1"
}

func deriveResolutionPreset(size string) string {
	if value, ok := sizeToResolution[size]; ok {
		return value
	}
	return "1k"
}

func buildAspectSizeSelection(aspect string, currentResolution string, apiMode string, requestPolicy string, imageModelID string) string {
	if aspect == "auto" {
		return "auto"
	}
	currentResolution = normalizeResolutionChoice(currentResolution, apiMode, requestPolicy, imageModelID)
	if currentResolution == "auto" {
		currentResolution = "1k"
	}
	return buildSizeSelection(aspect, currentResolution)
}

func buildResolutionSizeSelection(currentAspect string, resolution string, apiMode string, requestPolicy string, imageModelID string) string {
	if resolution == "auto" {
		return "auto"
	}
	resolution = normalizeResolutionChoice(resolution, apiMode, requestPolicy, imageModelID)
	if currentAspect == "auto" {
		currentAspect = "1:1"
	}
	return buildSizeSelection(currentAspect, resolution)
}

func buildSizeSelection(aspect string, resolution string) string {
	if aspect == "auto" || resolution == "auto" {
		return "auto"
	}
	if sizes, ok := sizeMatrix[aspect]; ok {
		if value, ok := sizes[resolution]; ok {
			return value
		}
	}
	return "1024x1024"
}

func visibleResolutionChoices(apiMode string, requestPolicy string, imageModelID string) []choice {
	if supportsExplicitLargeSizes(apiMode, requestPolicy, imageModelID) {
		return resolutionChoices
	}
	filtered := make([]choice, 0, 2)
	for _, item := range resolutionChoices {
		if item.Value == "2k" || item.Value == "4k" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func normalizeBatchCount(value int) int {
	if value < 1 {
		return 1
	}
	if value > 9 {
		return 9
	}
	return value
}

func choiceLabel(choices []choice, value string) string {
	for _, item := range choices {
		if item.Value == value {
			return item.Label
		}
	}
	return value
}

func classifyImageModel(modelID string) string {
	value := strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case strings.Contains(value, "gpt-image"):
		return "gpt-image"
	case strings.Contains(value, "dall-e-3"), strings.Contains(value, "dalle-3"), strings.Contains(value, "dall-e3"), strings.Contains(value, "dalle3"):
		return "dalle3"
	default:
		return "other"
	}
}

func supportsExplicitLargeSizes(apiMode string, requestPolicy string, imageModelID string) bool {
	family := classifyImageModel(imageModelID)
	if strings.TrimSpace(apiMode) == string(client.APIModeImages) {
		return family == "gpt-image" || family == "dalle3"
	}
	if family == "gpt-image" {
		return true
	}
	return strings.TrimSpace(requestPolicy) == string(client.RequestPolicyCompat)
}

func normalizeResolutionChoice(resolution string, apiMode string, requestPolicy string, imageModelID string) string {
	allowed := visibleResolutionChoices(apiMode, requestPolicy, imageModelID)
	for _, item := range allowed {
		if item.Value == resolution {
			return resolution
		}
	}
	return "1k"
}

func sizeCapabilityHint(apiMode string, requestPolicy string, imageModelID string) string {
	if supportsExplicitLargeSizes(apiMode, requestPolicy, imageModelID) {
		return ""
	}
	return "当前链路只保证基础尺寸稳定可用。"
}
