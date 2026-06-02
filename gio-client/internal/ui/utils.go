package ui

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	sharedCompat "image-studio/shared/compat"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"
)

func decodeImageB64(imageB64 string) (image.Image, error) {
	data, err := base64.StdEncoding.DecodeString(imageB64)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image bytes: %w", err)
	}
	return img, nil
}

func decodeImageFile(path string) (image.Image, error) {
	file, err := os.Open(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode image file: %w", err)
	}
	return img, nil
}

func appendBounded(logs []string, line string) []string {
	const maxLogs = 240
	logs = append(logs, time.Now().Format("15:04:05")+"  "+line)
	if len(logs) > maxLogs {
		copy(logs, logs[len(logs)-maxLogs:])
		logs = logs[:maxLogs]
	}
	return logs
}

func shortPrompt(prompt string) string {
	prompt = strings.Join(strings.Fields(prompt), " ")
	if len([]rune(prompt)) <= 40 {
		return prompt
	}
	runes := []rune(prompt)
	return string(runes[:40]) + "..."
}

func compactNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func localDayStart(now time.Time) time.Time {
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, now.Location())
}

func formatHistoryDay(createdAt int64) string {
	if createdAt <= 0 {
		return "更早"
	}
	return time.UnixMilli(createdAt).Format("2006-01-02")
}

func matchHistoryDate(createdAt int64, filter string, now time.Time) bool {
	switch strings.TrimSpace(filter) {
	case "", "all":
		return true
	case "today":
		return createdAt >= localDayStart(now).UnixMilli()
	case "week":
		return createdAt >= now.AddDate(0, 0, -7).UnixMilli()
	default:
		return true
	}
}

func matchHistoryQuery(item sharedCompat.HistoryItem, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return true
	}
	candidates := []string{
		item.Prompt,
		item.RevisedPrompt,
		item.SavedPath,
		item.Size,
		item.Quality,
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), query) {
			return true
		}
	}
	return false
}

func copyImageFile(src, dst string) (string, error) {
	src = strings.TrimSpace(src)
	dst = strings.TrimSpace(strings.Trim(dst, `"'`))
	if src == "" {
		return "", errors.New("源图片路径为空")
	}
	if dst == "" {
		return "", errors.New("目标路径为空")
	}
	if strings.HasSuffix(dst, string(os.PathSeparator)) || strings.HasSuffix(dst, "/") || strings.HasSuffix(dst, `\`) {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	if info, err := os.Stat(dst); err == nil && info.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	if filepath.Ext(dst) == "" {
		dst += filepath.Ext(src)
	}
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return "", err
	}
	if filepath.Clean(absSrc) == filepath.Clean(absDst) {
		return absDst, nil
	}
	if err := os.MkdirAll(filepath.Dir(absDst), 0o700); err != nil {
		return "", err
	}
	in, err := os.Open(absSrc)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(absDst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(absDst)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	ok = true
	return absDst, nil
}
