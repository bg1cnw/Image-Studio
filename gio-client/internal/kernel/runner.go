package kernel

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuanhua/image-gptcodex/pkg/client"
)

type Config struct {
	APIKey             string
	BaseURL            string
	TextModelID        string
	ImageModelID       string
	Prompt             string
	Mode               client.Mode
	APIMode            client.APIMode
	RequestPolicy      client.RequestPolicy
	ImagesNewAPICompat bool
	Size               string
	Quality            string
	OutputFormat       string
	ProxyMode          string
	ProxyURL           string
	SourcePaths        []string
	OutputDir          string
	Seed               int64
	NegativePrompt     string
	PartialImages      int
	StyleTag           string
	BatchIndex         int
}

type Callbacks struct {
	Log      func(string)
	Progress func(stage string, elapsed int, bytes int64)
	Partial  func(client.PartialImage)
}

type Result struct {
	SavedPath     string
	RawPath       string
	ImageB64      string
	RevisedPrompt string
	SourceEvent   string
}

type Runner struct{}

func DefaultConfig() Config {
	return Config{
		TextModelID:   client.TextModel,
		ImageModelID:  client.ImageModel,
		Mode:          client.ModeGenerate,
		APIMode:       client.APIModeResponses,
		RequestPolicy: client.RequestPolicyOpenAI,
		Size:          client.DefaultSize,
		Quality:       client.DefaultQuality,
		OutputFormat:  client.OutputFormat,
		ProxyMode:     client.ProxyModeSystem,
		OutputDir:     DefaultOutputDir(),
		PartialImages: client.DefaultPartialImages,
	}
}

func DefaultOutputDir() string {
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, "Pictures", "Image Studio")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "gio-images"
	}
	return filepath.Join(cwd, "gio-images")
}

func ParseSourcePaths(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t' || r == ','
	})
	paths := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, field := range fields {
		path := strings.TrimSpace(field)
		path = strings.Trim(path, `"`)
		path = strings.Trim(path, `'`)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func (Runner) Run(ctx context.Context, cfg Config, cb Callbacks) (Result, error) {
	cfg = normalizeConfig(cfg)
	if cfg.APIKey == "" {
		return Result{}, fmt.Errorf("API Key 不能为空")
	}
	if cfg.Prompt == "" {
		return Result{}, fmt.Errorf("提示词不能为空")
	}
	if cfg.BaseURL == "" {
		return Result{}, fmt.Errorf("上游 BASE_URL 不能为空")
	}
	if cfg.Mode == client.ModeEdit && len(cfg.SourcePaths) == 0 {
		return Result{}, fmt.Errorf("图生图模式需要至少一张源图")
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o700); err != nil {
		return Result{}, fmt.Errorf("创建输出目录失败:%w", err)
	}
	imagesDir := filepath.Join(cfg.OutputDir, "images")
	logDir := filepath.Join(cfg.OutputDir, "log")
	if err := os.MkdirAll(imagesDir, 0o700); err != nil {
		return Result{}, fmt.Errorf("创建图片目录失败:%w", err)
	}
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return Result{}, fmt.Errorf("创建日志目录失败:%w", err)
	}

	proxy, err := client.NormalizeProxyConfig(cfg.ProxyMode, cfg.ProxyURL)
	if err != nil {
		return Result{}, err
	}
	transport, err := client.PickTransportWithProxy(proxy)
	if err != nil {
		return Result{}, err
	}

	opts := client.Options{
		APIKey:             cfg.APIKey,
		BaseURL:            cfg.BaseURL,
		TextModelID:        cfg.TextModelID,
		ImageModelID:       cfg.ImageModelID,
		Prompt:             cfg.Prompt,
		Mode:               cfg.Mode,
		APIMode:            cfg.APIMode,
		RequestPolicy:      cfg.RequestPolicy,
		ImagesNewAPICompat: cfg.ImagesNewAPICompat,
		Size:               cfg.Size,
		Quality:            cfg.Quality,
		OutputFormat:       cfg.OutputFormat,
		Proxy:              proxy,
		Seed:               cfg.Seed,
		NegativePrompt:     cfg.NegativePrompt,
		PartialImages:      cfg.PartialImages,
	}
	if cfg.Mode == client.ModeEdit {
		if err := attachSourceImages(&opts, cfg.SourcePaths); err != nil {
			return Result{}, err
		}
	}

	timestamp := time.Now().Format("20060102-150405")
	result, rawPath, err := client.RequestAndExtractWithRetriesAndPartial(
		ctx,
		transport,
		opts,
		logDir,
		timestamp,
		nonNilLog(cb.Log),
		cb.Progress,
		cb.Partial,
	)
	if err != nil {
		return Result{RawPath: absPathOrRaw(rawPath)}, err
	}
	rawPath = absPathOrRaw(rawPath)

	imageName := buildImageName(cfg.Mode, cfg.Prompt, timestamp, cfg.OutputFormat)
	savedPath, err := saveImage(result.ImageB64, filepath.Join(imagesDir, imageName))
	if err != nil {
		return Result{RawPath: rawPath, ImageB64: result.ImageB64}, err
	}
	return Result{
		SavedPath:     savedPath,
		RawPath:       rawPath,
		ImageB64:      result.ImageB64,
		RevisedPrompt: result.RevisedPrompt,
		SourceEvent:   result.SourceEvent,
	}, nil
}

func absPathOrRaw(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func normalizeConfig(cfg Config) Config {
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.TextModelID = strings.TrimSpace(cfg.TextModelID)
	cfg.ImageModelID = strings.TrimSpace(cfg.ImageModelID)
	cfg.Prompt = strings.TrimSpace(cfg.Prompt)
	cfg.Size = strings.TrimSpace(cfg.Size)
	cfg.Quality = strings.TrimSpace(cfg.Quality)
	cfg.OutputFormat = strings.TrimSpace(cfg.OutputFormat)
	cfg.ProxyMode = strings.TrimSpace(cfg.ProxyMode)
	cfg.ProxyURL = strings.TrimSpace(cfg.ProxyURL)
	cfg.OutputDir = strings.TrimSpace(cfg.OutputDir)
	cfg.NegativePrompt = strings.TrimSpace(cfg.NegativePrompt)
	if cfg.TextModelID == "" {
		cfg.TextModelID = client.TextModel
	}
	if cfg.ImageModelID == "" {
		cfg.ImageModelID = client.ImageModel
	}
	if cfg.Mode != client.ModeEdit {
		cfg.Mode = client.ModeGenerate
	}
	if cfg.APIMode != client.APIModeImages {
		cfg.APIMode = client.APIModeResponses
	}
	if cfg.RequestPolicy != client.RequestPolicyCompat {
		cfg.RequestPolicy = client.RequestPolicyOpenAI
	}
	if cfg.Size == "" {
		cfg.Size = client.DefaultSize
	}
	if cfg.Quality == "" {
		cfg.Quality = client.DefaultQuality
	}
	if cfg.OutputFormat == "" {
		cfg.OutputFormat = client.OutputFormat
	}
	if cfg.ProxyMode == "" {
		cfg.ProxyMode = client.ProxyModeSystem
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = DefaultOutputDir()
	}
	if cfg.PartialImages <= 0 {
		cfg.PartialImages = client.DefaultPartialImages
	}
	cfg.SourcePaths = normalizeSourcePaths(cfg.SourcePaths)
	return cfg
}

func normalizeSourcePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, raw := range paths {
		path, err := client.NormalizePath(raw)
		if err != nil {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func attachSourceImages(opts *client.Options, paths []string) error {
	normalized := normalizeSourcePaths(paths)
	if len(normalized) == 0 {
		return fmt.Errorf("图生图模式需要至少一张源图")
	}
	if opts.APIMode == client.APIModeImages {
		opts.ImagePaths = normalized
		return nil
	}
	dataURLs := make([]string, 0, len(normalized))
	for _, path := range normalized {
		dataURL, err := client.ImageFileToDataURL(path)
		if err != nil {
			return err
		}
		dataURLs = append(dataURLs, dataURL)
	}
	opts.ImageDataURLs = dataURLs
	return nil
}

func buildImageName(mode client.Mode, prompt, timestamp, outputFormat string) string {
	prefix := "generate"
	if mode == client.ModeEdit {
		prefix = "edit"
	}
	slug := client.Slugify(prompt, "image")
	ext := client.FileExtForFormat(outputFormat)
	return fmt.Sprintf("image-%s-%s-%s.%s", prefix, slug, timestamp, ext)
}

func saveImage(imageB64, outputPath string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(imageB64)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}
	abs, err := filepath.Abs(outputPath)
	if err != nil {
		return outputPath, nil
	}
	return abs, nil
}

func nonNilLog(log func(string)) func(string) {
	if log != nil {
		return log
	}
	return func(string) {}
}
