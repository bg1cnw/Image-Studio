package client

import "errors"

const (
	BaseURL            = "https://gptcodex.top"
	TextModel          = "gpt-5.5"
	ImageModel         = "gpt-image-2"
	DefaultSize        = "1024x1024"
	DefaultQuality     = "auto"
	OutputFormat       = "png"
	MaxInputImageBytes = 50 * 1024 * 1024
	MaxAttempts        = 3
)

// Tunable knobs (exposed as vars so tests can shrink them).
var (
	RetryBackoffSeconds  = 15
	StatusIntervalSecond = 10
)

var SupportedImageMime = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
}

type SizeOption struct {
	Label string
	Value string
}

var SizeOptions = []SizeOption{
	{"正方形 1024x1024", "1024x1024"},
	{"横版 1536x1024", "1536x1024"},
	{"竖版 1024x1536", "1024x1536"},
	{"宽屏 2048x1152", "2048x1152"},
}

type QualityOption struct {
	Label string
	Value string
}

var QualityOptions = []QualityOption{
	{"标准 auto(推荐)", "auto"},
	{"高质量 high", "high"},
	{"中等 medium", "medium"},
	{"快速草稿 low", "low"},
}

// Mode is "generate" (text-to-image) or "edit" (image-to-image).
type Mode string

const (
	ModeGenerate Mode = "generate"
	ModeEdit     Mode = "edit"
)

// TransportKind selects HTTP implementation.
type TransportKind string

const (
	TransportAuto   TransportKind = "auto"
	TransportNative TransportKind = "native"
	TransportCurl   TransportKind = "curl"
)

// Options drives a single image request.
type Options struct {
	APIKey  string
	Prompt  string
	Mode    Mode
	Size    string
	Quality string

	// ImageDataURLs holds one or more data: URLs for edit / multi-reference
	// requests. When non-empty the request switches to "edit" action and each
	// URL becomes its own input_image content block (in order).
	ImageDataURLs []string

	// Deprecated: kept for callers that still pass a single source image.
	// If both are set, ImageDataURLs wins. Single URL is appended to the slice.
	ImageDataURL string

	MaskB64 string // optional, reserved for Phase 3 GUI; omitted from payload when empty

	// Seed pins the random source so users can reproduce a result. 0 means
	// "let the model pick", and the field is then omitted from the payload.
	Seed int64

	// Negative prompt — only included when non-empty. Whether the upstream
	// gptcodex-image tool reads it varies; sent for forward compatibility.
	NegativePrompt string

	// Optional overrides for the URL and model IDs. Empty values fall back
	// to BaseURL / TextModel / ImageModel constants.
	BaseURL         string
	TextModelID     string
	ImageModelID    string

	Transport TransportKind // auto | native | curl
}

// EffectiveImageDataURLs returns the merged list, deduplicating empty entries.
func (o Options) EffectiveImageDataURLs() []string {
	urls := make([]string, 0, len(o.ImageDataURLs)+1)
	for _, u := range o.ImageDataURLs {
		if u != "" {
			urls = append(urls, u)
		}
	}
	if o.ImageDataURL != "" {
		urls = append(urls, o.ImageDataURL)
	}
	return urls
}

// ImageResult is the extracted image payload.
type ImageResult struct {
	ImageB64      string
	RevisedPrompt string
	SourceEvent   string // "final" | "partial" | "json"
}

// Progress is streamed by Transport.Stream during a request.
type Progress struct {
	Stage   string // human-readable status, e.g. "图片正在生成"
	Elapsed int    // seconds since request start (filled by orchestrator, not Transport)
	Bytes   int64  // bytes received so far (filled by orchestrator)
}

// Request is what the transport actually sends.
type Request struct {
	URL     string
	APIKey  string
	Payload []byte // JSON-encoded payload (UTF-8)
}

// Sentinel errors so callers (and tests) can branch on cause.
var (
	ErrNoImageInResponse = errors.New("no image base64 in response")
	ErrEmptyPrompt       = errors.New("prompt must not be empty")
	ErrEmptyAPIKey       = errors.New("api key must not be empty")
)
