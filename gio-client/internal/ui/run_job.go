package ui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	gioCompat "image-studio/gio-client/internal/compat"
	"image-studio/gio-client/internal/kernel"
	sharedCompat "image-studio/shared/compat"

	"github.com/yuanhua/image-gptcodex/pkg/client"
)

func (a *App) startRun() {
	a.startRunWithConfig(a.currentConfig(), normalizeBatchCount(a.batchCount))
}

func (a *App) retryLastRun() {
	a.mu.Lock()
	cfg := a.lastRunConfig
	total := a.lastRunBatchCount
	ok := a.lastRunValid
	a.mu.Unlock()
	if !ok {
		return
	}
	a.startRunWithConfig(cfg, total)
}

func (a *App) startRunWithConfig(cfg kernel.Config, total int) {
	if a.isRunning() {
		return
	}
	total = normalizeBatchCount(total)
	a.rememberPrompt(cfg.Prompt)
	if err := gioCompat.SaveConfig(cfg); err != nil {
		a.appendLog("兼容配置保存失败: " + err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.running = true
	a.cancel = cancel
	a.lastRunConfig = cfg
	a.lastRunBatchCount = total
	a.lastRunValid = true
	a.lastErrorMessage = ""
	a.status = fmt.Sprintf("正在提交 1/%d", total)
	a.activePromptGroup = historyPromptGroup{}
	a.batchResultIDs = nil
	a.resultGridOpen = total > 1
	a.logs = appendBounded(a.logs, fmt.Sprintf("开始任务 1/%d: %s", total, shortPrompt(cfg.Prompt)))
	a.mu.Unlock()
	a.invalidateNow()

	go func() {
		batchStarted := time.Now()
		for i := 0; i < total; i++ {
			if err := ctx.Err(); err != nil {
				a.finishCancelled()
				return
			}
			jobCfg := cfg
			jobCfg.BatchIndex = i
			jobCfg.Prompt = augmentPromptWithStyle(cfg.Prompt, cfg.StyleTag)
			if cfg.Seed != 0 {
				jobCfg.Seed = cfg.Seed + int64(i)
			}
			jobLabel := fmt.Sprintf("%d/%d", i+1, total)
			jobStarted := time.Now()
			res, err := a.runner.Run(ctx, jobCfg, kernel.Callbacks{
				Log: func(line string) {
					a.appendLog("[" + jobLabel + "] " + line)
				},
				Progress: func(stage string, elapsed int, bytes int64) {
					a.setStatus(fmt.Sprintf("%s · %s · %ds · %s", jobLabel, stage, elapsed, client.FormatBytes(bytes)))
				},
				Partial: func(partial client.PartialImage) {
					a.setStatus(fmt.Sprintf("%s · 收到流式预览 #%d", jobLabel, partial.PartialImageIndex))
				},
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					a.finishCancelled()
					return
				}
				a.finishWithError(err, res.RawPath)
				return
			}
			img, err := decodeImageB64(res.ImageB64)
			if err != nil {
				a.finishWithError(err, res.RawPath)
				return
			}
			elapsedSec := time.Since(jobStarted).Seconds()
			if err := gioCompat.SaveConfigAndHistory(jobCfg, res, elapsedSec); err != nil {
				a.appendLog("兼容历史保存失败: " + err.Error())
			}
			compatState, _, _ := gioCompat.LoadState()
			compatState = sharedCompat.Normalize(compatState)
			selectedItem, hasSelected := historyItemBySavedPath(compatState.History, res.SavedPath)
			if !hasSelected {
				selectedItem, hasSelected = newestHistoryItem(compatState.History)
			}
			activeProfileID := ""
			if profile, ok := gioCompat.ActiveProfile(compatState); ok {
				activeProfileID = profile.ID
			}
			a.mu.Lock()
			a.status = fmt.Sprintf("完成 %s · %.1fs", jobLabel, time.Since(batchStarted).Seconds())
			a.lastErrorMessage = ""
			a.result = resultState{
				Image:         img,
				SavedPath:     res.SavedPath,
				RawPath:       res.RawPath,
				RevisedPrompt: res.RevisedPrompt,
				SourceEvent:   res.SourceEvent,
				Item:          selectedItem,
				HasItem:       hasSelected,
				Rev:           a.result.Rev + 1,
			}
			a.history = append([]sharedCompat.HistoryItem(nil), compatState.History...)
			a.profiles = append([]sharedCompat.UpstreamProfile(nil), compatState.Profiles...)
			a.activeProfileID = activeProfileID
			if hasSelected {
				a.selectedHistoryID = selectedItem.ID
			}
			if total > 1 {
				if hasSelected {
					a.batchResultIDs = append(a.batchResultIDs, selectedItem.ID)
				}
			}
			if !a.savePromptSuppressed && res.SavedPath != "" && total == 1 {
				a.savePromptVisible = true
				a.savePromptSourcePath = res.SavedPath
				a.savePromptPathInput.SetText(res.SavedPath)
			}
			a.logs = appendBounded(a.logs, fmt.Sprintf("生成完成 %s: %s", jobLabel, res.SavedPath))
			if i == total-1 {
				a.running = false
				a.cancel = nil
				a.status = fmt.Sprintf("完成 - %.1fs", time.Since(batchStarted).Seconds())
			}
			a.mu.Unlock()
			a.invalidateNow()
		}
	}()
}

func (a *App) currentConfig() kernel.Config {
	seed, _ := strconv.ParseInt(strings.TrimSpace(a.seedInput.Text()), 10, 64)
	partial, _ := strconv.Atoi(strings.TrimSpace(a.partialImagesInput.Text()))
	sourcePaths := kernel.ParseSourcePaths(a.sourcePathsInput.Text())
	if client.Mode(a.mode) == client.ModeEdit && len(sourcePaths) == 0 {
		if current := strings.TrimSpace(a.readSnapshot().Result.SavedPath); current != "" {
			sourcePaths = []string{current}
		}
	}
	return kernel.Config{
		APIKey:         a.apiKeyInput.Text(),
		BaseURL:        a.baseURLInput.Text(),
		TextModelID:    a.textModelInput.Text(),
		ImageModelID:   a.imageModelInput.Text(),
		Prompt:         a.promptInput.Text(),
		Mode:           client.Mode(a.mode),
		APIMode:        client.APIMode(a.api),
		RequestPolicy:  client.RequestPolicy(a.policy),
		Size:           a.size,
		Quality:        a.quality,
		OutputFormat:   a.format,
		ProxyMode:      a.proxy,
		ProxyURL:       a.proxyURLInput.Text(),
		SourcePaths:    sourcePaths,
		OutputDir:      a.outputDirInput.Text(),
		Seed:           seed,
		NegativePrompt: a.negativePromptInput.Text(),
		PartialImages:  partial,
		StyleTag:       a.styleTag,
	}
}
