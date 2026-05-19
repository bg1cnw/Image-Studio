// Package fsio centralizes filesystem helpers used by the CLI and Wails app.
package fsio

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuanhua/image-gptcodex/pkg/client"
)

// EnsureDir creates dir (and parents) if it doesn't exist.
func EnsureDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("output directory is empty")
	}
	return os.MkdirAll(dir, 0o755)
}

// SaveImage writes base64 PNG bytes to outputPath and returns the absolute path.
func SaveImage(imageB64, outputPath string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(imageB64)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}
	abs, err := filepath.Abs(outputPath)
	if err != nil {
		return outputPath, nil //nolint:nilerr
	}
	return abs, nil
}

// DefaultOutputDir returns the default place to write images.
// CLI uses CWD/images; this is overridable by the caller.
func DefaultOutputDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "images"
	}
	return filepath.Join(cwd, "images")
}

// BuildImageName composes the final image filename matching the Python script.
func BuildImageName(mode client.Mode, prompt, timestamp string) string {
	prefix := "generate"
	if mode == client.ModeEdit {
		prefix = "edit"
	}
	slug := client.Slugify(prompt, "image")
	return fmt.Sprintf("gptcodex-%s-%s-%s.%s", prefix, slug, timestamp, client.OutputFormat)
}
