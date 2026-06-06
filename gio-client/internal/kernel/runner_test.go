package kernel

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gen2brain/avif"
	"github.com/yuanhua/image-gptcodex/pkg/client"
)

func TestParseSourcePaths(t *testing.T) {
	got := ParseSourcePaths(" /tmp/a.png\n'/tmp/b.jpg',\"/tmp/a.png\" ")
	want := []string{"/tmp/a.png", "/tmp/b.jpg"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeConfigDefaults(t *testing.T) {
	cfg := normalizeConfig(Config{
		Prompt:    "  hello  ",
		Mode:      client.Mode("unknown"),
		OutputDir: filepath.Join("tmp", "out"),
	})
	if cfg.Prompt != "hello" {
		t.Fatalf("prompt=%q", cfg.Prompt)
	}
	if cfg.Mode != client.ModeGenerate {
		t.Fatalf("mode=%q", cfg.Mode)
	}
	if cfg.APIMode != client.APIModeResponses {
		t.Fatalf("api mode=%q", cfg.APIMode)
	}
	if cfg.TextModelID == "" || cfg.ImageModelID == "" || cfg.OutputFormat == "" {
		t.Fatalf("missing defaults: %#v", cfg)
	}
	if cfg.PartialImages != 0 {
		t.Fatalf("partial images=%d want 0", cfg.PartialImages)
	}
}

func TestNormalizeConfigPreservesZeroPartialImages(t *testing.T) {
	cfg := normalizeConfig(Config{
		Prompt:        "hello",
		OutputDir:     filepath.Join("tmp", "out"),
		PartialImages: 0,
	})
	if cfg.PartialImages != 0 {
		t.Fatalf("partial images=%d want 0", cfg.PartialImages)
	}
	cfg = normalizeConfig(Config{
		Prompt:        "hello",
		OutputDir:     filepath.Join("tmp", "out"),
		PartialImages: -1,
	})
	if cfg.PartialImages != client.DefaultPartialImages {
		t.Fatalf("negative partial images=%d want default %d", cfg.PartialImages, client.DefaultPartialImages)
	}
}

func TestBuildImageNameMapsJPEGExtension(t *testing.T) {
	got := buildImageName(client.ModeEdit, "A cat wearing sunglasses", "20260531-120000", "jpeg")
	want := "image-edit-a-cat-wearing-sunglasses-20260531-120000.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestProbeUpstreamReturnsModelCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer sk-test" {
			t.Fatalf("authorization=%q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.5"},{"id":"gpt-image-2"}]}`))
	}))
	defer server.Close()

	result, err := ProbeUpstream(context.Background(), Config{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("ProbeUpstream: %v", err)
	}
	if result.ModelCount != 2 {
		t.Fatalf("ModelCount=%d want 2", result.ModelCount)
	}
}

func TestSaveThumbnailCreatesDownscaledPNG(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	file, err := os.Create(sourcePath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 1800, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1800; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0x55, G: 0x99, B: 0xdd, A: 0xff})
		}
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode source: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}

	thumbPath, err := saveThumbnail(sourcePath, filepath.Join(dir, "thumb.png"), historyThumbMaxEdge)
	if err != nil {
		t.Fatalf("saveThumbnail: %v", err)
	}
	thumbFile, err := os.Open(thumbPath)
	if err != nil {
		t.Fatalf("open thumb: %v", err)
	}
	defer thumbFile.Close()
	thumb, _, err := image.Decode(thumbFile)
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if thumb.Bounds().Dx() > historyThumbMaxEdge || thumb.Bounds().Dy() > historyThumbMaxEdge {
		t.Fatalf("thumb bounds=%v exceed %d", thumb.Bounds(), historyThumbMaxEdge)
	}
}

func TestEnsurePreviewForPathWithFallbackUsesLegacyThumb(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "images", "source.png")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatalf("mkdir images: %v", err)
	}
	sourceFile, err := os.Create(sourcePath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	sourceImage := image.NewNRGBA(image.Rect(0, 0, 512, 384))
	for y := 0; y < 384; y++ {
		for x := 0; x < 512; x++ {
			sourceImage.SetNRGBA(x, y, color.NRGBA{R: 0xcc, G: 0x33, B: 0x22, A: 0xff})
		}
	}
	if err := png.Encode(sourceFile, sourceImage); err != nil {
		t.Fatalf("encode source: %v", err)
	}
	if err := sourceFile.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}

	legacyThumbPath := filepath.Join(dir, "thumbs", "source.avif")
	if err := os.MkdirAll(filepath.Dir(legacyThumbPath), 0o700); err != nil {
		t.Fatalf("mkdir thumbs: %v", err)
	}
	legacyThumbFile, err := os.Create(legacyThumbPath)
	if err != nil {
		t.Fatalf("create thumb: %v", err)
	}
	legacyThumbImage := image.NewNRGBA(image.Rect(0, 0, 96, 72))
	for y := 0; y < 72; y++ {
		for x := 0; x < 96; x++ {
			legacyThumbImage.SetNRGBA(x, y, color.NRGBA{R: 0x22, G: 0x66, B: 0xcc, A: 0xff})
		}
	}
	if err := avif.Encode(legacyThumbFile, legacyThumbImage, avif.Options{
		Quality:           100,
		QualityAlpha:      100,
		Speed:             1,
		ChromaSubsampling: image.YCbCrSubsampleRatio444,
	}); err != nil {
		t.Fatalf("encode legacy thumb: %v", err)
	}
	if err := legacyThumbFile.Close(); err != nil {
		t.Fatalf("close thumb: %v", err)
	}

	previewPath, err := EnsurePreviewForPathWithFallback(sourcePath, legacyThumbPath)
	if err != nil {
		t.Fatalf("EnsurePreviewForPathWithFallback: %v", err)
	}
	if got, want := previewPath, previewOutputPathForSource(sourcePath); got != want {
		t.Fatalf("previewPath=%q want %q", got, want)
	}
	previewFile, err := os.Open(previewPath)
	if err != nil {
		t.Fatalf("open preview: %v", err)
	}
	defer previewFile.Close()
	preview, _, err := image.Decode(previewFile)
	if err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	got := color.NRGBAModel.Convert(preview.At(0, 0)).(color.NRGBA)
	want := color.NRGBA{R: 0x22, G: 0x66, B: 0xcc, A: 0xff}
	if absDiffByte(got.R, want.R) > 2 || absDiffByte(got.G, want.G) > 2 || absDiffByte(got.B, want.B) > 2 || absDiffByte(got.A, want.A) > 0 {
		t.Fatalf("preview pixel=%#v want %#v", got, want)
	}
	if preview.Bounds().Dx() > historyPreviewMaxEdge || preview.Bounds().Dy() > historyPreviewMaxEdge {
		t.Fatalf("preview bounds=%v exceed %d", preview.Bounds(), historyPreviewMaxEdge)
	}
}

func absDiffByte(a byte, b byte) int {
	if a >= b {
		return int(a - b)
	}
	return int(b - a)
}

func TestEnsurePreviewForPathCreatesDownscaledPNG(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	file, err := os.Create(sourcePath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 1800, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1800; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0x33, G: 0x77, B: 0xbb, A: 0xff})
		}
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode source: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}

	previewPath, err := EnsurePreviewForPath(sourcePath)
	if err != nil {
		t.Fatalf("EnsurePreviewForPath: %v", err)
	}
	previewFile, err := os.Open(previewPath)
	if err != nil {
		t.Fatalf("open preview: %v", err)
	}
	defer previewFile.Close()
	preview, _, err := image.Decode(previewFile)
	if err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if preview.Bounds().Dx() > historyPreviewMaxEdge || preview.Bounds().Dy() > historyPreviewMaxEdge {
		t.Fatalf("preview bounds=%v exceed %d", preview.Bounds(), historyPreviewMaxEdge)
	}
}

func BenchmarkEnsurePreviewAndThumbForPathCurrent(b *testing.B) {
	dir := b.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	file, err := os.Create(sourcePath)
	if err != nil {
		b.Fatalf("create source: %v", err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 1800, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1800; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0x33, G: 0x77, B: 0xbb, A: 0xff})
		}
	}
	if err := png.Encode(file, img); err != nil {
		b.Fatalf("encode source: %v", err)
	}
	if err := file.Close(); err != nil {
		b.Fatalf("close source: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		previewPath := previewOutputPathForSource(sourcePath)
		thumbPath := thumbOutputPathForSource(sourcePath)
		_ = os.Remove(previewPath)
		_ = os.Remove(thumbPath)
		if _, _, err := EnsurePreviewAndThumbForPath(sourcePath); err != nil {
			b.Fatalf("EnsurePreviewAndThumbForPath: %v", err)
		}
	}
}

func BenchmarkEnsurePreviewAndThumbForPathLegacy(b *testing.B) {
	dir := b.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	file, err := os.Create(sourcePath)
	if err != nil {
		b.Fatalf("create source: %v", err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 1800, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1800; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0x33, G: 0x77, B: 0xbb, A: 0xff})
		}
	}
	if err := png.Encode(file, img); err != nil {
		b.Fatalf("encode source: %v", err)
	}
	if err := file.Close(); err != nil {
		b.Fatalf("close source: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		previewPath := previewOutputPathForSource(sourcePath)
		thumbPath := thumbOutputPathForSource(sourcePath)
		_ = os.Remove(previewPath)
		_ = os.Remove(thumbPath)
		if _, err := saveThumbnail(sourcePath, previewPath, historyPreviewMaxEdge); err != nil {
			b.Fatalf("saveThumbnail preview: %v", err)
		}
		if _, err := saveThumbnail(sourcePath, thumbPath, historyThumbMaxEdge); err != nil {
			b.Fatalf("saveThumbnail thumb: %v", err)
		}
	}
}

func BenchmarkEnsurePreviewAndThumbForPathCurrentThumbAlreadyExists(b *testing.B) {
	dir := b.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	file, err := os.Create(sourcePath)
	if err != nil {
		b.Fatalf("create source: %v", err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 1800, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1800; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0x22, G: 0x66, B: 0xaa, A: 0xff})
		}
	}
	if err := png.Encode(file, img); err != nil {
		b.Fatalf("encode source: %v", err)
	}
	if err := file.Close(); err != nil {
		b.Fatalf("close source: %v", err)
	}
	thumbPath, err := EnsureThumbForPath(sourcePath)
	if err != nil {
		b.Fatalf("EnsureThumbForPath: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		previewPath := previewOutputPathForSource(sourcePath)
		_ = os.Remove(previewPath)
		if _, _, err := EnsurePreviewAndThumbForPath(sourcePath); err != nil {
			b.Fatalf("EnsurePreviewAndThumbForPath: %v", err)
		}
	}
	_ = thumbPath
}

func BenchmarkEnsurePreviewAndThumbForPathLegacyThumbAlreadyExists(b *testing.B) {
	dir := b.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	file, err := os.Create(sourcePath)
	if err != nil {
		b.Fatalf("create source: %v", err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 1800, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1800; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0x22, G: 0x66, B: 0xaa, A: 0xff})
		}
	}
	if err := png.Encode(file, img); err != nil {
		b.Fatalf("encode source: %v", err)
	}
	if err := file.Close(); err != nil {
		b.Fatalf("close source: %v", err)
	}
	thumbPath, err := EnsureThumbForPath(sourcePath)
	if err != nil {
		b.Fatalf("EnsureThumbForPath: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		previewPath := previewOutputPathForSource(sourcePath)
		_ = os.Remove(previewPath)
		if _, err := saveThumbnail(sourcePath, previewPath, historyPreviewMaxEdge); err != nil {
			b.Fatalf("saveThumbnail preview: %v", err)
		}
		if _, err := os.Stat(thumbPath); err != nil {
			b.Fatalf("thumb missing: %v", err)
		}
	}
}
