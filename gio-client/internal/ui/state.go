package ui

import (
	"os"
	"strings"

	gioCompat "image-studio/gio-client/internal/compat"
	sharedCompat "image-studio/shared/compat"
)

func (a *App) saveCurrentConfig() {
	if err := gioCompat.SaveConfig(a.currentConfig()); err != nil {
		a.appendLog("兼容配置保存失败: " + err.Error())
	}
	if err := a.saveActiveProfileMetadata(); err != nil {
		a.appendLog("配置元数据保存失败: " + err.Error())
	}
}

func (a *App) cancelRun() {
	a.mu.Lock()
	cancel := a.cancel
	if cancel != nil {
		a.cancel = nil
		a.running = false
		a.status = "已取消"
		a.logs = appendBounded(a.logs, "任务已取消")
	}
	a.mu.Unlock()
	if cancel != nil {
		cancel()
		a.invalidateNow()
	}
}

func (a *App) finishWithError(err error, rawPath string) {
	a.mu.Lock()
	a.running = false
	a.cancel = nil
	a.status = "失败"
	a.lastErrorMessage = strings.TrimSpace(err.Error())
	if rawPath != "" {
		a.result.RawPath = rawPath
	}
	a.logs = appendBounded(a.logs, "失败: "+err.Error())
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) finishCancelled() {
	a.mu.Lock()
	a.running = false
	a.cancel = nil
	a.status = "已取消"
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) appendLog(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	a.mu.Lock()
	a.logs = appendBounded(a.logs, line)
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) setStatus(status string) {
	a.mu.Lock()
	a.status = status
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) clearLogs() {
	a.mu.Lock()
	a.logs = nil
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) closeSavePrompt() {
	a.mu.Lock()
	a.savePromptVisible = false
	a.savePromptSourcePath = ""
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) openHistoryTimeline() {
	a.mu.Lock()
	a.historyTimelineOpen = true
	a.historyTimelineModeFilter = a.historyModeFilter
	a.historyTimelineDateFilter = a.historyDateFilter
	a.historyTimelineQueryInput.SetText(a.historyQueryInput.Text())
	a.expandedPromptGroups = map[string]bool{}
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) closeHistoryTimeline() {
	a.mu.Lock()
	a.historyTimelineOpen = false
	a.expandedPromptGroups = map[string]bool{}
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) batchGridCountLocked() int {
	total := len(a.batchResultIDs)
	if a.running && a.lastRunBatchCount > total {
		total = a.lastRunBatchCount
	}
	return total
}

func (a *App) canOpenResultGridLocked() bool {
	return a.batchGridCountLocked() > 1
}

func (a *App) openResultGrid() {
	a.mu.Lock()
	if !a.canOpenResultGridLocked() {
		a.mu.Unlock()
		return
	}
	a.resultGridOpen = true
	a.compare = resultState{Rev: a.compare.Rev + 1}
	a.compareSplitSlider.Value = 0.5
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) closeResultGrid() {
	a.mu.Lock()
	a.resultGridOpen = false
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) openSavePromptForCurrent() {
	a.mu.Lock()
	src := strings.TrimSpace(a.result.SavedPath)
	a.mu.Unlock()
	a.openSavePromptForPath(src)
}

func (a *App) openSavePromptForPath(path string) {
	src := strings.TrimSpace(path)
	if src == "" {
		return
	}
	a.mu.Lock()
	a.savePromptVisible = true
	a.savePromptSourcePath = src
	a.savePromptPathInput.SetText(src)
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) setSavePromptSuppressed(value bool) {
	a.mu.Lock()
	a.savePromptSuppressed = value
	a.savePromptNeverAsk.Value = value
	a.mu.Unlock()
	if err := gioCompat.SetSavePromptSuppressed(value); err != nil {
		a.appendLog("保存提示设置失败: " + err.Error())
	}
	a.invalidateNow()
}

func (a *App) savePromptCopy() {
	a.mu.Lock()
	src := a.savePromptSourcePath
	dst := a.savePromptPathInput.Text()
	a.mu.Unlock()
	saved, err := copyImageFile(src, dst)
	if err != nil {
		a.appendLog("另存失败: " + err.Error())
		return
	}
	a.appendLog("已另存图片: " + saved)
	a.closeSavePrompt()
}

func (a *App) openRawResponseModal(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	content, err := os.ReadFile(path)
	text := ""
	readErr := ""
	if err != nil {
		readErr = err.Error()
	} else {
		text = string(content)
		const maxPreview = 200_000
		if len(text) > maxPreview {
			text = text[:maxPreview] + "\n\n... [截断,完整内容请查看文件]"
		}
	}
	a.mu.Lock()
	a.rawResponseModalPath = path
	a.rawResponseModalError = readErr
	a.rawResponseModalText = text
	a.rawResponseViewerInput.SetText(text)
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) closeRawResponseModal() {
	a.mu.Lock()
	a.rawResponseModalPath = ""
	a.rawResponseModalError = ""
	a.rawResponseModalText = ""
	a.rawResponseViewerInput.SetText("")
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) readSnapshot() snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	logs := append([]string(nil), a.logs...)
	history := append([]sharedCompat.HistoryItem(nil), a.history...)
	batchResults := historyItemsByIDs(history, a.batchResultIDs)
	profiles := append([]sharedCompat.UpstreamProfile(nil), a.profiles...)
	promptHistory := append([]string(nil), a.promptHistory...)
	presets := append([]sharedCompat.Preset(nil), a.presets...)
	return snapshot{
		Running:               a.running,
		Status:                a.status,
		Logs:                  logs,
		History:               history,
		BatchResults:          batchResults,
		BatchTotal:            a.lastRunBatchCount,
		Profiles:              profiles,
		ActiveProfileID:       a.activeProfileID,
		SelectedHistoryID:     a.selectedHistoryID,
		PromptHistory:         promptHistory,
		Presets:               presets,
		OptimizingPrompt:      a.optimizingPrompt,
		TestingUpstream:       a.testingUpstream,
		LastProbeSummary:      a.lastProbeSummary,
		ActivePromptGroup:     a.activePromptGroup,
		ActiveResultDetail:    a.activeResultDetail,
		HistoryTimelineOpen:   a.historyTimelineOpen,
		Fullscreen:            a.fullscreen,
		LastErrorMessage:      a.lastErrorMessage,
		LastRunAvailable:      a.lastRunValid,
		RawResponseModalPath:  a.rawResponseModalPath,
		RawResponseModalText:  a.rawResponseModalText,
		RawResponseModalError: a.rawResponseModalError,
		ResultGridOpen:        a.resultGridOpen,
		Compare:               a.compare,
		CompareSplit:          a.compareSplitSlider.Value,
		Result:                a.result,
		SavePromptVisible:     a.savePromptVisible,
	}
}

func (a *App) dismissFailureState() {
	a.mu.Lock()
	a.lastErrorMessage = ""
	if a.status == "失败" {
		if a.result.HasItem {
			a.status = "已载入历史结果"
		} else {
			a.status = "准备就绪"
		}
	}
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) isRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func (a *App) invalidateNow() {
	if a.invalidate != nil {
		a.invalidate()
	}
}
