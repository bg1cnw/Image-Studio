package backend

import (
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMediaHandlerServesRegisteredFullAndAVIFThumb(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root, err := defaultOutputDir()
	if err != nil {
		t.Fatal(err)
	}
	imagesDir := imagesSubdir(root)
	thumbsDir := thumbsSubdir(root)
	if err := os.MkdirAll(imagesDir, secureDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(thumbsDir, secureDirMode); err != nil {
		t.Fatal(err)
	}
	fullPath := filepath.Join(imagesDir, "sample.png")
	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, secureFileMode)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 640, 320))
	for y := 0; y < 320; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 160, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	thumbPath := filepath.Join(thumbsDir, "sample.avif")
	tw, th, err := createAVIFThumbnail(fullPath, thumbPath, 384)
	if err != nil {
		t.Fatal(err)
	}
	if tw != 384 || th != 192 {
		t.Fatalf("thumbnail size = %dx%d, want 384x192", tw, th)
	}

	svc := NewService()
	ref, err := svc.RegisterMediaAsset(fullPath, thumbPath)
	if err != nil {
		t.Fatal(err)
	}
	if ref.ImageID == "" || ref.PreviewURL == "" || ref.FullURL == "" {
		t.Fatalf("incomplete media ref: %+v", ref)
	}
	handler := svc.MediaHandler(http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodGet, ref.PreviewURL, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("thumb status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/avif" {
		t.Fatalf("thumb content-type = %q", got)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("empty thumb body")
	}

	req = httptest.NewRequest(http.MethodGet, ref.FullURL, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("full status = %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("empty full body")
	}
}
