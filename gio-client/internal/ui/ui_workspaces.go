package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"image-studio/gio-client/internal/kernel"
	sharedCompat "image-studio/shared/compat"

	"gioui.org/widget"
)

func (a *App) initWorkspaces() {
	ws := workspaceState{
		ID:   fmt.Sprintf("ws-%d", time.Now().UnixNano()),
		Name: "图片 1",
	}
	a.workspaces = []workspaceState{ws}
	a.activeWorkspaceID = ws.ID
	a.saveActiveWorkspaceSnapshot()
}

func (a *App) workspaceButton(id string) *widget.Clickable {
	if a.workspaceButtons == nil {
		a.workspaceButtons = map[string]*widget.Clickable{}
	}
	if btn, ok := a.workspaceButtons[id]; ok {
		return btn
	}
	btn := new(widget.Clickable)
	a.workspaceButtons[id] = btn
	return btn
}

func (a *App) closeWorkspaceButton(id string) *widget.Clickable {
	if a.closeWorkspaceButtons == nil {
		a.closeWorkspaceButtons = map[string]*widget.Clickable{}
	}
	if btn, ok := a.closeWorkspaceButtons[id]; ok {
		return btn
	}
	btn := new(widget.Clickable)
	a.closeWorkspaceButtons[id] = btn
	return btn
}

func (a *App) displayedWorkspaceName(ws workspaceState) string {
	name := strings.TrimSpace(ws.Name)
	if name == "" {
		name = "未命名"
	}
	if ws.ID == a.activeWorkspaceID && isDefaultWorkspaceName(name) {
		name = conciseWorkspaceName(a.promptInput.Text(), name)
	}
	return name
}

func (a *App) buildWorkspaceSnapshot() workspaceState {
	name := "未命名"
	for _, ws := range a.workspaces {
		if ws.ID == a.activeWorkspaceID {
			name = ws.Name
			break
		}
	}
	if isDefaultWorkspaceName(name) && strings.TrimSpace(a.promptInput.Text()) != "" {
		name = conciseWorkspaceName(a.promptInput.Text(), name)
	}
	return workspaceState{
		ID:                  a.activeWorkspaceID,
		Name:                name,
		Prompt:              a.promptInput.Text(),
		NegativePrompt:      a.negativePromptInput.Text(),
		Mode:                a.mode,
		Size:                a.size,
		Quality:             a.quality,
		OutputFormat:        a.format,
		StyleTag:            a.styleTag,
		SeedText:            a.seedInput.Text(),
		BatchCount:          a.batchCount,
		SourcePathsText:     a.sourcePathsInput.Text(),
		ResultSavedPath:     a.result.SavedPath,
		ResultRawPath:       a.result.RawPath,
		ResultRevisedPrompt: a.result.RevisedPrompt,
		ResultSourceEvent:   a.result.SourceEvent,
		ResultItem:          a.result.Item,
		ResultHasItem:       a.result.HasItem,
		SelectedHistoryID:   a.selectedHistoryID,
		BatchResultIDs:      append([]string(nil), a.batchResultIDs...),
		ResultGridOpen:      a.resultGridOpen,
	}
}

func (a *App) saveActiveWorkspaceSnapshot() {
	if strings.TrimSpace(a.activeWorkspaceID) == "" {
		return
	}
	snapshot := a.buildWorkspaceSnapshot()
	next := make([]workspaceState, 0, len(a.workspaces))
	found := false
	for _, ws := range a.workspaces {
		if ws.ID == snapshot.ID {
			next = append(next, snapshot)
			found = true
			continue
		}
		next = append(next, ws)
	}
	if !found {
		next = append(next, snapshot)
	}
	a.workspaces = next
}

func (a *App) applyWorkspace(ws workspaceState) {
	a.promptInput.SetText(ws.Prompt)
	a.negativePromptInput.SetText(ws.NegativePrompt)
	a.mode = ws.Mode
	a.size = ws.Size
	a.quality = ws.Quality
	a.format = ws.OutputFormat
	a.styleTag = ws.StyleTag
	a.seedInput.SetText(ws.SeedText)
	a.batchCount = normalizeBatchCount(ws.BatchCount)
	a.sourcePathsInput.SetText(ws.SourcePathsText)
	a.selectedHistoryID = ws.SelectedHistoryID
	a.activePromptGroup = historyPromptGroup{}
	a.batchResultIDs = append([]string(nil), ws.BatchResultIDs...)
	a.resultGridOpen = ws.ResultGridOpen && len(ws.BatchResultIDs) > 1
	a.promptHelperOpen = false
	a.settingsModalOpen = false
	a.activeResultDetail = sharedCompat.HistoryItem{}
	a.result = resultState{
		SavedPath:     ws.ResultSavedPath,
		RawPath:       ws.ResultRawPath,
		RevisedPrompt: ws.ResultRevisedPrompt,
		SourceEvent:   ws.ResultSourceEvent,
		Item:          ws.ResultItem,
		HasItem:       ws.ResultHasItem,
		Rev:           a.result.Rev + 1,
	}
	if strings.TrimSpace(ws.ResultSavedPath) != "" {
		if img, err := a.imageForPath(ws.ResultSavedPath); err == nil {
			a.result.Image = img
		}
	}
	a.invalidateNow()
}

func (a *App) createWorkspace() {
	if a.isRunning() {
		a.appendLog("运行中不能新建标签")
		return
	}
	a.commitWorkspaceRename()
	a.saveActiveWorkspaceSnapshot()
	name := fmt.Sprintf("图片 %d", len(a.workspaces)+1)
	ws := workspaceState{
		ID:             fmt.Sprintf("ws-%d", time.Now().UnixNano()),
		Name:           name,
		Mode:           string(kernel.DefaultConfig().Mode),
		Size:           kernel.DefaultConfig().Size,
		Quality:        kernel.DefaultConfig().Quality,
		OutputFormat:   kernel.DefaultConfig().OutputFormat,
		BatchCount:     1,
		ResultGridOpen: false,
	}
	a.workspaces = append(a.workspaces, ws)
	a.activeWorkspaceID = ws.ID
	a.applyWorkspace(ws)
}

func (a *App) switchWorkspace(id string) {
	if a.isRunning() {
		a.appendLog("运行中不能切换标签")
		return
	}
	if a.workspaceRenameID != "" && a.workspaceRenameID != id {
		a.commitWorkspaceRename()
	}
	if id == a.activeWorkspaceID {
		return
	}
	a.saveActiveWorkspaceSnapshot()
	for _, ws := range a.workspaces {
		if ws.ID == id {
			a.activeWorkspaceID = id
			a.applyWorkspace(ws)
			return
		}
	}
}

func (a *App) closeWorkspace(id string) {
	if len(a.workspaces) <= 1 {
		a.appendLog("至少保留一个标签")
		return
	}
	if a.isRunning() {
		a.appendLog("运行中不能关闭标签")
		return
	}
	if a.workspaceRenameID == id {
		a.cancelWorkspaceRename()
	}
	a.saveActiveWorkspaceSnapshot()
	next := make([]workspaceState, 0, len(a.workspaces)-1)
	for _, ws := range a.workspaces {
		if ws.ID == id {
			continue
		}
		next = append(next, ws)
	}
	a.workspaces = next
	if a.activeWorkspaceID == id && len(next) > 0 {
		a.activeWorkspaceID = next[0].ID
		a.applyWorkspace(next[0])
	}
}

func (a *App) startWorkspaceRename(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	for _, ws := range a.workspaces {
		if ws.ID != id {
			continue
		}
		a.workspaceRenameID = id
		a.workspaceNameInput.SetText(strings.TrimSpace(ws.Name))
		a.invalidateNow()
		return
	}
}

func (a *App) commitWorkspaceRename() {
	id := strings.TrimSpace(a.workspaceRenameID)
	if id == "" {
		return
	}
	name := strings.TrimSpace(a.workspaceNameInput.Text())
	if name == "" {
		name = "未命名"
	}
	next := make([]workspaceState, 0, len(a.workspaces))
	for _, ws := range a.workspaces {
		if ws.ID == id {
			ws.Name = name
		}
		next = append(next, ws)
	}
	a.workspaces = next
	a.workspaceRenameID = ""
	a.workspaceNameInput.SetText("")
	a.invalidateNow()
}

func (a *App) cancelWorkspaceRename() {
	a.workspaceRenameID = ""
	a.workspaceNameInput.SetText("")
	a.invalidateNow()
}

func (a *App) handleWorkspacePrimaryClick(id string, now time.Time) {
	if strings.TrimSpace(id) == "" {
		return
	}
	if a.workspaceRenameID == id {
		return
	}
	if a.workspaceLastClickID == id && !a.workspaceLastClickAt.IsZero() && now.Sub(a.workspaceLastClickAt) <= 450*time.Millisecond {
		a.workspaceLastClickID = ""
		a.workspaceLastClickAt = time.Time{}
		a.startWorkspaceRename(id)
		return
	}
	a.workspaceLastClickID = id
	a.workspaceLastClickAt = now
	a.switchWorkspace(id)
}

func conciseWorkspaceName(prompt string, fallback string) string {
	prompt = strings.Join(strings.Fields(strings.TrimSpace(prompt)), " ")
	if prompt == "" {
		return fallback
	}
	runes := []rune(prompt)
	if len(runes) > 18 {
		runes = runes[:18]
	}
	return string(runes)
}

func isDefaultWorkspaceName(name string) bool {
	if !strings.HasPrefix(name, "图片 ") {
		return false
	}
	_, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(name, "图片 ")))
	return err == nil
}
