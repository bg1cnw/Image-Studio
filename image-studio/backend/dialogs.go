package backend

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// OpenImageDialog shows a file picker filtered to supported image types and
// returns the selected absolute path along with its byte size.
func (s *Service) OpenImageDialog() (SelectFileResponse, error) {
	path, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择源图片",
		Filters: []runtime.FileFilter{
			{DisplayName: "支持的图片 (*.png;*.jpg;*.jpeg;*.webp)", Pattern: "*.png;*.jpg;*.jpeg;*.webp"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return SelectFileResponse{}, err
	}
	if path == "" {
		return SelectFileResponse{}, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return SelectFileResponse{}, err
	}
	return SelectFileResponse{Path: path, Size: info.Size()}, nil
}

// SaveImageAs prompts the user for a destination and writes the base64 PNG to disk.
func (s *Service) SaveImageAs(imageB64, suggestedName string) (string, error) {
	if suggestedName == "" {
		suggestedName = fmt.Sprintf("image-%d.png", time.Now().Unix())
	}
	dst, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "保存图片",
		DefaultFilename: suggestedName,
		Filters: []runtime.FileFilter{
			{DisplayName: "PNG 图片 (*.png)", Pattern: "*.png"},
		},
	})
	if err != nil || dst == "" {
		return "", err
	}
	return writeBase64PNG(imageB64, dst)
}

// GetOutputDir returns the directory where generated images and raw response
// dumps are written.
func (s *Service) GetOutputDir() (string, error) {
	return defaultOutputDir()
}

// OpenOutputDir reveals the output directory in the OS file explorer.
func (s *Service) OpenOutputDir() error {
	dir, err := defaultOutputDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return openInExplorer(dir)
}

// OpenExternalURL launches a URL in the default browser. Used for GitHub /
// MIT license / Issues links in the About dialog and Footer.
func (s *Service) OpenExternalURL(url string) error {
	if url == "" {
		return errors.New("url is empty")
	}
	return openInExplorer(url)
}

// ReadImageAsBase64 loads an image file from disk and returns its bytes as
// standard base64. Used by the frontend to refresh the canvas after a
// rotate/flip/crop operation produced a new file in imports/.
func (s *Service) ReadImageAsBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ReadTextFile returns a file's contents as a string. Used to display the raw
// SSE response in the "查看 raw" modal.
func (s *Service) ReadTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ExportHistoryToFile writes a JSON dump (provided by the frontend) to a
// user-chosen path. Powers the "导出历史" action in settings.
func (s *Service) ExportHistoryToFile(jsonContent string) (string, error) {
	dst, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "导出历史记录",
		DefaultFilename: fmt.Sprintf("image-studio-history-%s.json", time.Now().Format("20060102-150405")),
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || dst == "" {
		return "", err
	}
	if err := os.WriteFile(dst, []byte(jsonContent), 0o644); err != nil {
		return "", err
	}
	return dst, nil
}

// ImportHistoryFromFile opens a file picker and returns the JSON content as a
// string. The frontend then parses and merges the entries into IndexedDB.
func (s *Service) ImportHistoryFromFile() (string, error) {
	src, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择历史 JSON 文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || src == "" {
		return "", err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
