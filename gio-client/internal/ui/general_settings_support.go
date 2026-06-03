package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gioCompat "image-studio/gio-client/internal/compat"
	sharedCompat "image-studio/shared/compat"
)

var generalKernelRuntimeChoices = []settingsOptionChoice{
	{Title: "auto(按宿主自动选择)", Detail: "按宿主自动选择", Value: "auto"},
	{Title: "local(桌面 Go/Wails)", Detail: "桌面 Go/Wails", Value: "local"},
	{Title: "remote(共享远程内核)", Detail: "共享远程内核", Value: "remote"},
}

type historyExportPayload struct {
	Version    int                        `json:"version"`
	ExportedAt string                     `json:"exportedAt"`
	Count      int                        `json:"count"`
	Items      []sharedCompat.HistoryItem `json:"items"`
}

func normalizeKernelRuntimeMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "auto", "local", "remote":
		return strings.TrimSpace(mode)
	default:
		return "auto"
	}
}

func kernelRuntimeModeLabel(mode string) string {
	mode = normalizeKernelRuntimeMode(mode)
	for _, choice := range generalKernelRuntimeChoices {
		if choice.Value == mode {
			return choice.Title
		}
	}
	return generalKernelRuntimeChoices[0].Title
}

func (a *App) exportHistoryJSON() {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("导出历史失败: " + err.Error())
		return
	}
	state = sharedCompat.Normalize(state)
	if len(state.History) == 0 {
		a.appendLog("没有可导出的历史记录")
		return
	}
	filename := fmt.Sprintf("image-studio-history-%s.json", time.Now().Format("20060102-150405"))
	dst, err := chooseSaveJSONFile(filename)
	if err != nil {
		a.appendLog("选择导出文件失败: " + err.Error())
		return
	}
	if strings.TrimSpace(dst) == "" {
		return
	}
	payload := historyExportPayload{
		Version:    1,
		ExportedAt: time.Now().Format(time.RFC3339),
		Count:      len(state.History),
		Items:      append([]sharedCompat.HistoryItem(nil), state.History...),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		a.appendLog("导出历史失败: " + err.Error())
		return
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		a.appendLog("写入历史导出文件失败: " + err.Error())
		return
	}
	a.appendLog(fmt.Sprintf("已导出 %d 条历史: %s", len(state.History), filepath.Base(dst)))
}

func (a *App) importHistoryJSON() {
	src, err := chooseJSONFile()
	if err != nil {
		a.appendLog("选择历史文件失败: " + err.Error())
		return
	}
	if strings.TrimSpace(src) == "" {
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		a.appendLog("读取历史文件失败: " + err.Error())
		return
	}
	incoming, err := parseImportedHistoryItems(data)
	if err != nil {
		a.appendLog("导入历史失败: " + err.Error())
		return
	}
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("导入历史失败: " + err.Error())
		return
	}
	state = sharedCompat.Normalize(state)
	existing := make(map[string]struct{}, len(state.History))
	for _, item := range state.History {
		existing[strings.TrimSpace(item.ID)] = struct{}{}
	}
	added := 0
	for _, item := range incoming {
		item = normalizeImportedHistoryItem(item)
		if item.ID == "" || item.CreatedAt == 0 {
			continue
		}
		if _, ok := existing[item.ID]; ok {
			continue
		}
		existing[item.ID] = struct{}{}
		state.History = append(state.History, item)
		added++
	}
	if added == 0 {
		a.appendLog("导入完成，但没有新增历史项")
		return
	}
	sort.Slice(state.History, func(i, j int) bool {
		return state.History[i].CreatedAt > state.History[j].CreatedAt
	})
	state.UpdatedAt = time.Now().UnixMilli()
	if err := gioCompat.SaveState(state); err != nil {
		a.appendLog("保存导入后的历史失败: " + err.Error())
		return
	}
	a.mu.Lock()
	a.history = append([]sharedCompat.HistoryItem(nil), state.History...)
	a.mu.Unlock()
	if latest, ok := newestHistoryItem(state.History); ok {
		if err := a.loadHistoryPreview(latest, false); err != nil && !isMissingPreview(err) {
			a.appendLog("载入导入后的最近历史失败: " + err.Error())
		}
	}
	a.appendLog(fmt.Sprintf("已导入 %d 条历史: %s", added, filepath.Base(src)))
}

func parseImportedHistoryItems(data []byte) ([]sharedCompat.HistoryItem, error) {
	var payload historyExportPayload
	if err := json.Unmarshal(data, &payload); err == nil && len(payload.Items) > 0 {
		return payload.Items, nil
	}
	var items []sharedCompat.HistoryItem
	if err := json.Unmarshal(data, &items); err == nil && len(items) > 0 {
		return items, nil
	}
	return nil, fmt.Errorf("文件里没有可导入的历史记录")
}

func normalizeImportedHistoryItem(item sharedCompat.HistoryItem) sharedCompat.HistoryItem {
	item.ID = strings.TrimSpace(item.ID)
	item.Prompt = strings.TrimSpace(item.Prompt)
	item.RevisedPrompt = strings.TrimSpace(item.RevisedPrompt)
	item.Mode = strings.TrimSpace(item.Mode)
	item.Size = strings.TrimSpace(item.Size)
	item.Quality = strings.TrimSpace(item.Quality)
	item.OutputFormat = strings.TrimSpace(item.OutputFormat)
	item.SavedPath = strings.TrimSpace(item.SavedPath)
	item.ThumbPath = strings.TrimSpace(item.ThumbPath)
	if item.SavedPath == "" && item.ThumbPath != "" {
		item.SavedPath = item.ThumbPath
	}
	return item
}
