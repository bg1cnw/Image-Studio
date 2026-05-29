package backend

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/avif"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder for generated/imported images
)

const (
	mediaThumbMaxEdge = 384
	mediaAVIFQuality  = 46
	mediaAVIFAlphaQ   = 50
	mediaAVIFSpeed    = 8
)

type mediaAsset struct {
	ID            string
	FullPath      string
	ThumbPath     string
	FullURL       string
	PreviewURL    string
	PreviewWidth  int
	PreviewHeight int
}

func thumbsSubdir(root string) string { return filepath.Join(root, "thumbs") }

func (s *Service) registerGeneratedMedia(fullPath, thumbPath string, width, height int) (mediaAsset, error) {
	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return mediaAsset{}, err
	}
	thumbAbs := ""
	if strings.TrimSpace(thumbPath) != "" {
		thumbAbs, err = filepath.Abs(thumbPath)
		if err != nil {
			return mediaAsset{}, err
		}
	}
	id := mediaIDForPath(fullAbs)
	asset := mediaAsset{
		ID:            id,
		FullPath:      fullAbs,
		ThumbPath:     thumbAbs,
		FullURL:       "/media/full/" + id,
		PreviewURL:    "/media/thumb/" + id,
		PreviewWidth:  width,
		PreviewHeight: height,
	}
	s.mu.Lock()
	if s.mediaAssets == nil {
		s.mediaAssets = map[string]mediaAsset{}
	}
	s.mediaAssets[id] = asset
	s.mu.Unlock()
	return asset, nil
}

func (s *Service) RegisterMediaAsset(savedPath, thumbPath string) (MediaAssetRef, error) {
	allowedFull, err := s.ensureManagedReadablePath(savedPath, managedImageFile)
	if err != nil {
		return MediaAssetRef{}, err
	}
	allowedThumb := ""
	if strings.TrimSpace(thumbPath) != "" {
		if allowedThumb, err = s.ensureManagedReadablePath(thumbPath, managedImageFile); err != nil {
			allowedThumb = ""
		}
	}
	width, height := 0, 0
	if allowedThumb != "" {
		if cfg, cfgErr := imageConfig(allowedThumb); cfgErr == nil {
			width, height = cfg.Width, cfg.Height
		}
	}
	asset, err := s.registerGeneratedMedia(allowedFull, allowedThumb, width, height)
	if err != nil {
		return MediaAssetRef{}, err
	}
	return MediaAssetRef{
		ImageID:       asset.ID,
		SavedPath:     asset.FullPath,
		ThumbPath:     asset.ThumbPath,
		PreviewURL:    asset.PreviewURL,
		FullURL:       asset.FullURL,
		PreviewWidth:  asset.PreviewWidth,
		PreviewHeight: asset.PreviewHeight,
	}, nil
}

func mediaIDForPath(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:16])
}

func (s *Service) mediaAssetSnapshot(id string) (mediaAsset, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	asset, ok := s.mediaAssets[id]
	return asset, ok
}

func (s *Service) MediaHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/media/") {
			s.serveMedia(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) serveMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	kind, id, ok := parseMediaPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	asset, ok := s.mediaAssetSnapshot(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var path string
	switch kind {
	case "thumb":
		path = asset.ThumbPath
	case "full":
		path = asset.FullPath
	default:
		http.NotFound(w, r)
		return
	}
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if _, err := s.ensureManagedReadablePath(path, managedImageFile); err != nil {
		http.NotFound(w, r)
		return
	}
	setMediaHeaders(w, path, kind)
	http.ServeFile(w, r, path)
}

func parseMediaPath(path string) (kind string, id string, ok bool) {
	trimmed := strings.Trim(strings.TrimPrefix(path, "/media/"), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	if parts[0] != "thumb" && parts[0] != "full" {
		return "", "", false
	}
	if len(parts[1]) != 32 {
		return "", "", false
	}
	for _, r := range parts[1] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return "", "", false
		}
	}
	return parts[0], parts[1], true
}

func setMediaHeaders(w http.ResponseWriter, path, kind string) {
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if kind == "thumb" || strings.EqualFold(filepath.Ext(path), ".avif") {
		w.Header().Set("Content-Type", "image/avif")
	}
}

func createAVIFThumbnail(sourcePath, thumbPath string, maxEdge int) (width int, height int, err error) {
	if maxEdge <= 0 {
		maxEdge = mediaThumbMaxEdge
	}
	src, err := loadImage(sourcePath)
	if err != nil {
		return 0, 0, err
	}
	bounds := src.Bounds()
	sw, sh := bounds.Dx(), bounds.Dy()
	if sw <= 0 || sh <= 0 {
		return 0, 0, errors.New("invalid image dimensions")
	}
	scale := float64(maxEdge) / float64(max(sw, sh))
	if scale > 1 {
		scale = 1
	}
	tw := max(1, int(float64(sw)*scale+0.5))
	th := max(1, int(float64(sh)*scale+0.5))
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Src, nil)

	if err := os.MkdirAll(filepath.Dir(thumbPath), secureDirMode); err != nil {
		return 0, 0, err
	}
	f, err := os.OpenFile(thumbPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, secureFileMode)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	if err := avif.Encode(f, dst, avif.Options{
		Quality:           mediaAVIFQuality,
		QualityAlpha:      mediaAVIFAlphaQ,
		Speed:             mediaAVIFSpeed,
		ChromaSubsampling: image.YCbCrSubsampleRatio420,
	}); err != nil {
		return 0, 0, fmt.Errorf("encode avif thumbnail: %w", err)
	}
	return tw, th, nil
}

func imageConfig(path string) (image.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return image.Config{}, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	return cfg, err
}
