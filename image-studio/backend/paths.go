package backend

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuanhua/image-gptcodex/pkg/client"
)

// defaultOutputDir is where generated images and raw SSE dumps land.
// Falls back to ./images if the user config dir is unavailable.
func defaultOutputDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", "images"), nil
	}
	return filepath.Join(cfg, "image-studio", "images"), nil
}

// importsDir holds files dropped/pasted into the canvas, plus rotation/flip/
// crop derivatives. Separate from `images/` so the user can manage them apart.
func importsDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", "imports"), nil
	}
	return filepath.Join(cfg, "image-studio", "imports"), nil
}

// writeBase64PNG decodes a base64 image and writes it atomically; returns the
// absolute path of the written file.
func writeBase64PNG(b64, path string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	abs, _ := filepath.Abs(path)
	return abs, nil
}

// buildImageName composes the canonical filename for a generated image, e.g.
// `gptcodex-generate-cyberpunk-cat-20260518-210500.png`.
func buildImageName(mode client.Mode, prompt, timestamp string) string {
	prefix := "generate"
	if mode == client.ModeEdit {
		prefix = "edit"
	}
	slug := client.Slugify(prompt, "image")
	return fmt.Sprintf("gptcodex-%s-%s-%s.%s", prefix, slug, timestamp, client.OutputFormat)
}
