package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
	"time"

	gioCompat "image-studio/gio-client/internal/compat"
	"image-studio/gio-client/internal/kernel"
	sharedCompat "image-studio/shared/compat"

	"gioui.org/app"
	"gioui.org/widget"
	"github.com/yuanhua/image-gptcodex/pkg/client"
)

var errMissingPreview = errors.New("missing preview image")

func isMissingPreview(err error) bool {
	return errors.Is(err, errMissingPreview)
}

func newestHistoryItem(items []sharedCompat.HistoryItem) (sharedCompat.HistoryItem, bool) {
	if len(items) == 0 {
		return sharedCompat.HistoryItem{}, false
	}
	return items[0], true
}

func historyItemBySavedPath(items []sharedCompat.HistoryItem, savedPath string) (sharedCompat.HistoryItem, bool) {
	savedPath = strings.TrimSpace(savedPath)
	if savedPath == "" {
		return sharedCompat.HistoryItem{}, false
	}
	for _, item := range items {
		if strings.TrimSpace(item.SavedPath) == savedPath {
			return item, true
		}
	}
	return sharedCompat.HistoryItem{}, false
}

func historyItemByID(items []sharedCompat.HistoryItem, id string) (sharedCompat.HistoryItem, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return sharedCompat.HistoryItem{}, false
	}
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return sharedCompat.HistoryItem{}, false
}

func historyCounts(items []sharedCompat.HistoryItem) (generate int, edit int) {
	for _, item := range items {
		if item.Mode == "edit" {
			edit++
			continue
		}
		generate++
	}
	return generate, edit
}

func todayHistoryCount(items []sharedCompat.HistoryItem, now time.Time) int {
	start := localDayStart(now)
	count := 0
	for _, item := range items {
		if item.CreatedAt >= start.UnixMilli() {
			count++
		}
	}
	return count
}

func (a *App) profileButton(id string) *widget.Clickable {
	if a.profileButtons == nil {
		a.profileButtons = map[string]*widget.Clickable{}
	}
	if btn, ok := a.profileButtons[id]; ok {
		return btn
	}
	btn := new(widget.Clickable)
	a.profileButtons[id] = btn
	return btn
}

func (a *App) historyButton(id string) *widget.Clickable {
	if a.historyButtons == nil {
		a.historyButtons = map[string]*widget.Clickable{}
	}
	if btn, ok := a.historyButtons[id]; ok {
		return btn
	}
	btn := new(widget.Clickable)
	a.historyButtons[id] = btn
	return btn
}

func (a *App) settingsProfileButton(id string) *widget.Clickable {
	if a.settingsProfileButtons == nil {
		a.settingsProfileButtons = map[string]*widget.Clickable{}
	}
	if btn, ok := a.settingsProfileButtons[id]; ok {
		return btn
	}
	btn := new(widget.Clickable)
	a.settingsProfileButtons[id] = btn
	return btn
}

func (a *App) promptButton(id string) *widget.Clickable {
	if a.promptButtons == nil {
		a.promptButtons = map[string]*widget.Clickable{}
	}
	if btn, ok := a.promptButtons[id]; ok {
		return btn
	}
	btn := new(widget.Clickable)
	a.promptButtons[id] = btn
	return btn
}

func (a *App) sourceButton(id string) *widget.Clickable {
	if a.sourceButtons == nil {
		a.sourceButtons = map[string]*widget.Clickable{}
	}
	if btn, ok := a.sourceButtons[id]; ok {
		return btn
	}
	btn := new(widget.Clickable)
	a.sourceButtons[id] = btn
	return btn
}

func (a *App) historyActionButton(id string) *widget.Clickable {
	if a.historyActionButtons == nil {
		a.historyActionButtons = map[string]*widget.Clickable{}
	}
	if btn, ok := a.historyActionButtons[id]; ok {
		return btn
	}
	btn := new(widget.Clickable)
	a.historyActionButtons[id] = btn
	return btn
}

func (a *App) filteredHistory(items []sharedCompat.HistoryItem) []sharedCompat.HistoryItem {
	return filteredHistoryItems(
		items,
		a.historyQueryInput.Text(),
		a.historyModeFilter,
		a.historyDateFilter,
		time.Now(),
	)
}

func (a *App) filteredTimelineHistory(items []sharedCompat.HistoryItem) []sharedCompat.HistoryItem {
	return filteredHistoryItems(
		items,
		a.historyTimelineQueryInput.Text(),
		a.historyTimelineModeFilter,
		a.historyTimelineDateFilter,
		time.Now(),
	)
}

func (a *App) loadHistoryPreview(item sharedCompat.HistoryItem, addLog bool) error {
	img, err := a.imageForHistoryThumb(item)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.result = resultState{
		Image:         img,
		SavedPath:     item.SavedPath,
		RawPath:       item.RawPath,
		RevisedPrompt: item.RevisedPrompt,
		SourceEvent:   "history",
		Item:          item,
		HasItem:       item.ID != "",
		Rev:           a.result.Rev + 1,
	}
	a.compare = resultState{Rev: a.compare.Rev + 1}
	a.compareSplitSlider.Value = 0.5
	a.selectedHistoryID = item.ID
	a.status = "已载入历史结果"
	if addLog {
		a.logs = appendBounded(a.logs, "载入历史结果: "+shortPrompt(item.Prompt))
	}
	a.mu.Unlock()
	a.invalidateNow()
	return nil
}

func (a *App) imageForHistoryItem(item sharedCompat.HistoryItem) (image.Image, error) {
	return a.imageForHistorySource(item, false)
}

func (a *App) imageForHistoryThumb(item sharedCompat.HistoryItem) (image.Image, error) {
	return a.imageForHistorySource(item, true)
}

func (a *App) imageForHistorySource(item sharedCompat.HistoryItem, preferThumb bool) (image.Image, error) {
	cacheKey := historyImageCacheKey(item, preferThumb)
	if cached, ok := a.imageCache[cacheKey]; ok {
		if cached.Failed {
			return nil, errMissingPreview
		}
		return cached.Image, nil
	}

	load := func() (image.Image, error) {
		if strings.TrimSpace(item.ImageB64) != "" {
			return decodeImageB64(item.ImageB64)
		}
		paths := historyImagePaths(item, preferThumb)
		if len(paths) == 0 {
			return nil, errMissingPreview
		}
		var lastErr error
		for _, path := range paths {
			img, err := a.imageForPath(path)
			if err == nil {
				return img, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = errMissingPreview
		}
		return nil, lastErr
	}

	img, err := load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, errMissingPreview) {
			a.imageCache[cacheKey] = cachedImage{Failed: true}
			return nil, fmt.Errorf("%w: %v", errMissingPreview, err)
		}
		a.imageCache[cacheKey] = cachedImage{Failed: true}
		return nil, err
	}
	a.imageCache[cacheKey] = cachedImage{Image: img}
	return img, nil
}

func historyImagePaths(item sharedCompat.HistoryItem, preferThumb bool) []string {
	thumbPath := strings.TrimSpace(item.ThumbPath)
	savedPath := strings.TrimSpace(item.SavedPath)
	paths := make([]string, 0, 2)
	if preferThumb {
		if thumbPath != "" {
			paths = append(paths, thumbPath)
		}
		if savedPath != "" && savedPath != thumbPath {
			paths = append(paths, savedPath)
		}
		return paths
	}
	if savedPath != "" {
		paths = append(paths, savedPath)
	}
	if thumbPath != "" && thumbPath != savedPath {
		paths = append(paths, thumbPath)
	}
	return paths
}

func historyImageCacheKey(item sharedCompat.HistoryItem, preferThumb bool) string {
	if paths := historyImagePaths(item, preferThumb); len(paths) > 0 {
		prefix := "history-full:"
		if preferThumb {
			prefix = "history-thumb:"
		}
		return prefix + strings.Join(paths, "|")
	}
	mode := "full"
	if preferThumb {
		mode = "thumb"
	}
	if strings.TrimSpace(item.ID) != "" {
		return "history:" + mode + ":" + item.ID
	}
	if strings.TrimSpace(item.ImageB64) != "" {
		return "history:" + mode + ":inline"
	}
	return "history:" + mode + ":missing"
}

func (a *App) imageForPath(path string) (image.Image, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errMissingPreview
	}
	cacheKey := "path:" + path
	if cached, ok := a.imageCache[cacheKey]; ok {
		if cached.Failed {
			return nil, errMissingPreview
		}
		return cached.Image, nil
	}
	img, err := decodeImageFile(path)
	if err != nil {
		a.imageCache[cacheKey] = cachedImage{Failed: true}
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %v", errMissingPreview, err)
		}
		return nil, err
	}
	a.imageCache[cacheKey] = cachedImage{Image: img}
	return img, nil
}

func (a *App) switchActiveProfile(profileID string) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return
	}
	a.saveCurrentConfig()
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("读取上游配置失败: " + err.Error())
		return
	}
	state = sharedCompat.Normalize(state)
	if _, ok := historyItemByID(state.History, a.selectedHistoryID); !ok && len(state.History) > 0 {
		a.selectedHistoryID = state.History[0].ID
	}
	found := false
	for _, profile := range state.Profiles {
		if profile.ID == profileID {
			found = true
			break
		}
	}
	if !found {
		return
	}
	state.ActiveProfile = profileID
	state.UpdatedAt = time.Now().UnixMilli()
	if err := gioCompat.SaveState(state); err != nil {
		a.appendLog("切换上游失败: " + err.Error())
		return
	}
	cfg := gioCompat.ConfigFromState(kernel.DefaultConfig(), state)
	a.applyRuntimeConfig(cfg)
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.activeProfileID = profileID
	a.settingsSelectedProfileID = profileID
	a.status = "已切换上游: " + activeProfileName(state.Profiles, profileID)
	a.logs = appendBounded(a.logs, "切换上游配置: "+activeProfileName(state.Profiles, profileID))
	a.mu.Unlock()
	a.profileNameInput.SetText(activeProfileName(state.Profiles, profileID))
	if limit := activeProfileConcurrencyLimit(state.Profiles, profileID); limit > 0 {
		a.concurrencyLimitInput.SetText(strconv.Itoa(limit))
	} else {
		a.concurrencyLimitInput.SetText("")
	}
	a.profilePickerOpen = false
	a.invalidateNow()
}

func activeProfileName(profiles []sharedCompat.UpstreamProfile, profileID string) string {
	for _, profile := range profiles {
		if profile.ID == profileID {
			return strings.TrimSpace(profile.Name)
		}
	}
	return ""
}

func activeProfileAPIMode(profiles []sharedCompat.UpstreamProfile, profileID string) string {
	for _, profile := range profiles {
		if profile.ID == profileID {
			return strings.TrimSpace(profile.APIMode)
		}
	}
	return ""
}

func activeProfileConcurrencyLimit(profiles []sharedCompat.UpstreamProfile, profileID string) int {
	for _, profile := range profiles {
		if profile.ID == profileID {
			return profile.ConcurrencyLimit
		}
	}
	return 0
}

func profileByID(profiles []sharedCompat.UpstreamProfile, profileID string) (sharedCompat.UpstreamProfile, bool) {
	for _, profile := range profiles {
		if profile.ID == profileID {
			return profile, true
		}
	}
	return sharedCompat.UpstreamProfile{}, false
}

func normalizeProfileAPIMode(mode string) string {
	if strings.TrimSpace(mode) == string(client.APIModeImages) {
		return string(client.APIModeImages)
	}
	return string(client.APIModeResponses)
}

func normalizeProfilePolicy(policy string) string {
	if strings.TrimSpace(policy) == string(client.RequestPolicyCompat) {
		return string(client.RequestPolicyCompat)
	}
	return string(client.RequestPolicyOpenAI)
}

func normalizeSettingsSelectedProfileID(state sharedCompat.State, profileID string) string {
	profileID = strings.TrimSpace(profileID)
	if profileID != "" {
		if _, ok := profileByID(state.Profiles, profileID); ok {
			return profileID
		}
	}
	if strings.TrimSpace(state.ActiveProfile) != "" {
		if _, ok := profileByID(state.Profiles, state.ActiveProfile); ok {
			return state.ActiveProfile
		}
	}
	if len(state.Profiles) > 0 {
		return state.Profiles[0].ID
	}
	return ""
}

func (a *App) settingsDraftReady() bool {
	return strings.TrimSpace(a.baseURLInput.Text()) != "" && strings.TrimSpace(a.apiKeyInput.Text()) != ""
}

func (a *App) applySettingsProfileDraft(state sharedCompat.State, profile sharedCompat.UpstreamProfile) {
	a.settingsSelectedProfileID = profile.ID
	a.profileNameInput.SetText(strings.TrimSpace(profile.Name))
	a.api = normalizeProfileAPIMode(profile.APIMode)
	a.policy = normalizeProfilePolicy(profile.RequestPolicy)
	a.imagesNewAPICompat = profile.ImagesNewAPICompat
	a.baseURLInput.SetText(strings.TrimSpace(profile.BaseURL))
	a.textModelInput.SetText(strings.TrimSpace(profile.TextModelID))
	a.imageModelInput.SetText(strings.TrimSpace(profile.ImageModelID))
	if profile.ConcurrencyLimit > 0 {
		a.concurrencyLimitInput.SetText(strconv.Itoa(profile.ConcurrencyLimit))
	} else {
		a.concurrencyLimitInput.SetText("")
	}
	key, _ := gioCompat.ReadAPIKey(profile.ID)
	a.apiKeyInput.SetText(key)
	a.proxy = strings.TrimSpace(state.Settings.ProxyMode)
	if a.proxy == "" {
		a.proxy = client.ProxyModeSystem
	}
	a.proxyURLInput.SetText(strings.TrimSpace(state.Settings.ProxyURL))
	outputDir := strings.TrimSpace(state.Settings.OutputDir)
	if outputDir == "" {
		outputDir = kernel.DefaultOutputDir()
	}
	a.outputDirInput.SetText(outputDir)
	a.background = strings.TrimSpace(state.Settings.Background)
	if a.background == "" {
		a.background = client.DefaultBackground
	}
	if state.Settings.OutputCompression != nil {
		a.outputCompressionInput.SetText(strconv.Itoa(*state.Settings.OutputCompression))
	} else {
		a.outputCompressionInput.SetText(strconv.Itoa(client.DefaultOutputCompression))
	}
	a.inputFidelity = strings.TrimSpace(state.Settings.InputFidelity)
	if a.inputFidelity == "" {
		a.inputFidelity = client.DefaultInputFidelity
	}
	a.imageStyle = strings.TrimSpace(state.Settings.ImageStyle)
	if a.imageStyle == "" {
		a.imageStyle = client.DefaultImageStyle
	}
	a.moderation = strings.TrimSpace(state.Settings.Moderation)
	if a.moderation == "" {
		a.moderation = client.DefaultModeration
	}
	a.userIdentifierInput.SetText(strings.TrimSpace(state.Settings.UserIdentifier))
	if state.Settings.PartialImages != nil {
		a.partialImagesInput.SetText(strconv.Itoa(*state.Settings.PartialImages))
	} else {
		a.partialImagesInput.SetText(strconv.Itoa(client.DefaultPartialImages))
	}
	a.apiKeyVisible = false
}

func (a *App) loadSettingsProfileDraft(profileID string) error {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		return err
	}
	state = sharedCompat.Normalize(state)
	selectedID := normalizeSettingsSelectedProfileID(state, profileID)
	if selectedID == "" {
		a.settingsSelectedProfileID = ""
		a.profileNameInput.SetText("")
		a.baseURLInput.SetText("")
		a.apiKeyInput.SetText("")
		a.textModelInput.SetText(client.TextModel)
		a.imageModelInput.SetText(client.ImageModel)
		a.concurrencyLimitInput.SetText("")
		a.proxy = strings.TrimSpace(state.Settings.ProxyMode)
		if a.proxy == "" {
			a.proxy = client.ProxyModeSystem
		}
		a.proxyURLInput.SetText(strings.TrimSpace(state.Settings.ProxyURL))
		outputDir := strings.TrimSpace(state.Settings.OutputDir)
		if outputDir == "" {
			outputDir = kernel.DefaultOutputDir()
		}
		a.outputDirInput.SetText(outputDir)
		background := strings.TrimSpace(state.Settings.Background)
		if background == "" {
			background = client.DefaultBackground
		}
		a.background = background
		if state.Settings.OutputCompression != nil {
			a.outputCompressionInput.SetText(strconv.Itoa(*state.Settings.OutputCompression))
		} else {
			a.outputCompressionInput.SetText(strconv.Itoa(client.DefaultOutputCompression))
		}
		inputFidelity := strings.TrimSpace(state.Settings.InputFidelity)
		if inputFidelity == "" {
			inputFidelity = client.DefaultInputFidelity
		}
		a.inputFidelity = inputFidelity
		imageStyle := strings.TrimSpace(state.Settings.ImageStyle)
		if imageStyle == "" {
			imageStyle = client.DefaultImageStyle
		}
		a.imageStyle = imageStyle
		moderation := strings.TrimSpace(state.Settings.Moderation)
		if moderation == "" {
			moderation = client.DefaultModeration
		}
		a.moderation = moderation
		a.userIdentifierInput.SetText(strings.TrimSpace(state.Settings.UserIdentifier))
		if state.Settings.PartialImages != nil {
			a.partialImagesInput.SetText(strconv.Itoa(*state.Settings.PartialImages))
		} else {
			a.partialImagesInput.SetText(strconv.Itoa(client.DefaultPartialImages))
		}
		a.imagesNewAPICompat = false
		a.api = string(client.APIModeResponses)
		a.policy = string(client.RequestPolicyOpenAI)
		a.apiKeyVisible = false
		return nil
	}
	profile, ok := profileByID(state.Profiles, selectedID)
	if !ok {
		return nil
	}
	a.applySettingsProfileDraft(state, profile)
	return nil
}

func (a *App) restoreActiveRuntimeConfig(logErrors bool) error {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		if logErrors {
			a.appendLog("读取上游配置失败: " + err.Error())
		}
		return err
	}
	state = sharedCompat.Normalize(state)
	cfg := gioCompat.ConfigFromState(kernel.DefaultConfig(), state)
	a.applyRuntimeConfig(cfg)
	a.profileNameInput.SetText(activeProfileName(state.Profiles, state.ActiveProfile))
	if limit := activeProfileConcurrencyLimit(state.Profiles, state.ActiveProfile); limit > 0 {
		a.concurrencyLimitInput.SetText(strconv.Itoa(limit))
	} else {
		a.concurrencyLimitInput.SetText("")
	}
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.activeProfileID = state.ActiveProfile
	a.settingsSelectedProfileID = state.ActiveProfile
	a.mu.Unlock()
	a.apiKeyVisible = false
	return nil
}

func (a *App) openSettingsModal() {
	a.settingsModalOpen = true
	a.settingsHelpOpen = false
	if err := a.loadSettingsProfileDraft(a.activeProfileID); err != nil {
		a.appendLog("读取上游配置失败: " + err.Error())
	}
	a.invalidateNow()
}

func (a *App) closeSettingsModal() {
	a.settingsModalOpen = false
	a.settingsHelpOpen = false
	_ = a.restoreActiveRuntimeConfig(false)
	a.invalidateNow()
}

func (a *App) saveSettingsSelection() error {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		return err
	}
	state = sharedCompat.Normalize(state)
	selectedID := normalizeSettingsSelectedProfileID(state, a.settingsSelectedProfileID)
	if selectedID == "" {
		return nil
	}
	now := time.Now().UnixMilli()
	updated := false
	for i := range state.Profiles {
		if state.Profiles[i].ID != selectedID {
			continue
		}
		name := strings.TrimSpace(a.profileNameInput.Text())
		if name == "" {
			name = strings.TrimSpace(state.Profiles[i].Name)
		}
		if name == "" {
			name = nextProfileName(state.Profiles)
		}
		concurrencyLimit := 0
		if raw := strings.TrimSpace(a.concurrencyLimitInput.Text()); raw != "" {
			if value, err := strconv.Atoi(raw); err == nil && value > 0 {
				concurrencyLimit = value
			}
		}
		state.Profiles[i].Name = name
		state.Profiles[i].APIMode = normalizeProfileAPIMode(a.api)
		state.Profiles[i].RequestPolicy = normalizeProfilePolicy(a.policy)
		state.Profiles[i].ImagesNewAPICompat = a.imagesNewAPICompat
		state.Profiles[i].BaseURL = strings.TrimSpace(a.baseURLInput.Text())
		state.Profiles[i].TextModelID = strings.TrimSpace(a.textModelInput.Text())
		state.Profiles[i].ImageModelID = strings.TrimSpace(a.imageModelInput.Text())
		state.Profiles[i].ConcurrencyLimit = concurrencyLimit
		updated = true
		break
	}
	if !updated {
		return nil
	}
	state.Settings.ProxyMode = strings.TrimSpace(a.proxy)
	if state.Settings.ProxyMode == "" {
		state.Settings.ProxyMode = client.ProxyModeSystem
	}
	state.Settings.ProxyURL = strings.TrimSpace(a.proxyURLInput.Text())
	state.Settings.OutputDir = strings.TrimSpace(a.outputDirInput.Text())
	state.Settings.Background = strings.TrimSpace(a.background)
	outputCompression := client.DefaultOutputCompression
	if raw := strings.TrimSpace(a.outputCompressionInput.Text()); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			outputCompression = value
		}
	}
	state.Settings.OutputCompression = &outputCompression
	state.Settings.InputFidelity = strings.TrimSpace(a.inputFidelity)
	state.Settings.ImageStyle = strings.TrimSpace(a.imageStyle)
	state.Settings.Moderation = strings.TrimSpace(a.moderation)
	state.Settings.UserIdentifier = strings.TrimSpace(a.userIdentifierInput.Text())
	partialImages := client.DefaultPartialImages
	if raw := strings.TrimSpace(a.partialImagesInput.Text()); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			partialImages = value
		}
	}
	state.Settings.PartialImages = &partialImages
	state.UpdatedAt = now
	if err := gioCompat.SaveState(state); err != nil {
		return err
	}
	if err := gioCompat.WriteAPIKey(selectedID, a.apiKeyInput.Text()); err != nil {
		return err
	}
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.settingsSelectedProfileID = selectedID
	a.status = "已保存配置: " + activeProfileName(state.Profiles, selectedID)
	a.logs = appendBounded(a.logs, "已保存配置: "+activeProfileName(state.Profiles, selectedID))
	a.mu.Unlock()
	if selectedID == state.ActiveProfile {
		_ = a.restoreActiveRuntimeConfig(false)
	}
	return nil
}

func (a *App) activateStoredProfile(profileID string) error {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		return err
	}
	state = sharedCompat.Normalize(state)
	selectedID := normalizeSettingsSelectedProfileID(state, profileID)
	if selectedID == "" {
		return nil
	}
	name := activeProfileName(state.Profiles, selectedID)
	state.ActiveProfile = selectedID
	for i := range state.Profiles {
		if state.Profiles[i].ID == selectedID {
			state.Profiles[i].LastUsedAt = time.Now().UnixMilli()
			break
		}
	}
	state.UpdatedAt = time.Now().UnixMilli()
	if err := gioCompat.SaveState(state); err != nil {
		return err
	}
	if err := a.restoreActiveRuntimeConfig(false); err != nil {
		return err
	}
	a.mu.Lock()
	a.status = "已切换上游: " + name
	a.logs = appendBounded(a.logs, "切换上游配置: "+name)
	a.mu.Unlock()
	a.profilePickerOpen = false
	return nil
}

func (a *App) createSettingsProfile(apiMode string) error {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		return err
	}
	state = sharedCompat.Normalize(state)
	now := time.Now().UnixMilli()
	profileID := fmt.Sprintf("gio-%d", now)
	profile := sharedCompat.UpstreamProfile{
		ID:                 profileID,
		Name:               nextProfileName(state.Profiles),
		APIMode:            normalizeProfileAPIMode(apiMode),
		RequestPolicy:      string(client.RequestPolicyOpenAI),
		ImagesNewAPICompat: false,
		TextModelID:        client.TextModel,
		ImageModelID:       client.ImageModel,
		CreatedAt:          now,
		LastUsedAt:         now,
	}
	state.Profiles = append(state.Profiles, profile)
	activate := len(state.Profiles) == 1 || strings.TrimSpace(state.ActiveProfile) == ""
	if activate {
		state.ActiveProfile = profileID
	}
	state.UpdatedAt = now
	if err := gioCompat.SaveState(state); err != nil {
		return err
	}
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	if activate {
		a.activeProfileID = profileID
	}
	a.settingsSelectedProfileID = profileID
	a.status = "已创建配置: " + profile.Name
	a.logs = appendBounded(a.logs, "已创建配置: "+profile.Name)
	a.mu.Unlock()
	if activate {
		if err := a.restoreActiveRuntimeConfig(false); err != nil {
			return err
		}
	} else if err := a.loadSettingsProfileDraft(profileID); err != nil {
		return err
	}
	return nil
}

func (a *App) duplicateSettingsProfile() error {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		return err
	}
	state = sharedCompat.Normalize(state)
	selectedID := normalizeSettingsSelectedProfileID(state, a.settingsSelectedProfileID)
	current, ok := profileByID(state.Profiles, selectedID)
	if !ok {
		return nil
	}
	now := time.Now().UnixMilli()
	profileID := fmt.Sprintf("gio-%d", now)
	clone := current
	clone.ID = profileID
	clone.Name = nextProfileName(state.Profiles)
	clone.CreatedAt = now
	clone.LastUsedAt = now
	state.Profiles = append(state.Profiles, clone)
	state.UpdatedAt = now
	if key, _ := gioCompat.ReadAPIKey(current.ID); strings.TrimSpace(key) != "" {
		_ = gioCompat.WriteAPIKey(profileID, key)
	}
	if err := gioCompat.SaveState(state); err != nil {
		return err
	}
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.settingsSelectedProfileID = profileID
	a.status = "已复制配置: " + clone.Name
	a.logs = appendBounded(a.logs, "已复制配置: "+clone.Name)
	a.mu.Unlock()
	return a.loadSettingsProfileDraft(profileID)
}

func (a *App) deleteSettingsProfile() error {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		return err
	}
	state = sharedCompat.Normalize(state)
	selectedID := normalizeSettingsSelectedProfileID(state, a.settingsSelectedProfileID)
	current, ok := profileByID(state.Profiles, selectedID)
	if !ok {
		return nil
	}
	nextProfiles := make([]sharedCompat.UpstreamProfile, 0, len(state.Profiles)-1)
	for _, profile := range state.Profiles {
		if profile.ID == current.ID {
			continue
		}
		nextProfiles = append(nextProfiles, profile)
	}
	state.Profiles = nextProfiles
	if current.ID == state.ActiveProfile {
		if len(nextProfiles) > 0 {
			state.ActiveProfile = nextProfiles[0].ID
		} else {
			state.ActiveProfile = ""
		}
	}
	state.UpdatedAt = time.Now().UnixMilli()
	_ = gioCompat.WriteAPIKey(current.ID, "")
	if err := gioCompat.SaveState(state); err != nil {
		return err
	}
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.activeProfileID = state.ActiveProfile
	a.status = "已删除配置: " + current.Name
	a.logs = appendBounded(a.logs, "已删除配置: "+current.Name)
	a.mu.Unlock()
	nextSelectedID := ""
	if len(state.Profiles) > 0 {
		nextSelectedID = state.Profiles[0].ID
	}
	if state.ActiveProfile != "" {
		_ = a.restoreActiveRuntimeConfig(false)
	}
	return a.loadSettingsProfileDraft(nextSelectedID)
}

func (a *App) saveActiveProfileMetadata() error {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		return err
	}
	state = sharedCompat.Normalize(state)
	current, ok := currentActiveProfile(state)
	if !ok {
		return nil
	}
	name := strings.TrimSpace(a.profileNameInput.Text())
	if name == "" {
		name = strings.TrimSpace(current.Name)
	}
	if name == "" {
		name = nextProfileName(state.Profiles)
	}
	concurrencyLimit := 0
	if raw := strings.TrimSpace(a.concurrencyLimitInput.Text()); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			concurrencyLimit = value
		}
	}
	updated := false
	for i := range state.Profiles {
		if state.Profiles[i].ID != current.ID {
			continue
		}
		state.Profiles[i].Name = name
		state.Profiles[i].ConcurrencyLimit = concurrencyLimit
		state.Profiles[i].LastUsedAt = time.Now().UnixMilli()
		updated = true
		break
	}
	if !updated {
		return nil
	}
	state.UpdatedAt = time.Now().UnixMilli()
	if err := gioCompat.SaveState(state); err != nil {
		return err
	}
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.mu.Unlock()
	return nil
}

func filteredHistoryItems(
	items []sharedCompat.HistoryItem,
	query string,
	modeFilter string,
	dateFilter string,
	now time.Time,
) []sharedCompat.HistoryItem {
	query = strings.TrimSpace(strings.ToLower(query))
	modeFilter = strings.TrimSpace(modeFilter)
	dateFilter = strings.TrimSpace(dateFilter)
	if query == "" && modeFilter == "all" && dateFilter == "all" {
		return items
	}
	filtered := make([]sharedCompat.HistoryItem, 0, len(items))
	for _, item := range items {
		if modeFilter != "" && modeFilter != "all" && item.Mode != modeFilter {
			continue
		}
		if !matchHistoryDate(item.CreatedAt, dateFilter, now) {
			continue
		}
		if !matchHistoryQuery(item, query) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func nextProfileName(profiles []sharedCompat.UpstreamProfile) string {
	used := map[int]struct{}{}
	for _, profile := range profiles {
		raw := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(profile.Name), "配置"))
		n, err := strconv.Atoi(raw)
		if err == nil && n > 0 {
			used[n] = struct{}{}
		}
	}
	for i := 1; ; i++ {
		if _, ok := used[i]; !ok {
			return "配置" + strconv.Itoa(i)
		}
	}
}

func (a *App) createBlankProfile() {
	a.createBlankProfileWithMode(string(client.APIModeResponses))
}

func (a *App) createBlankProfileWithMode(apiMode string) {
	a.saveCurrentConfig()
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("创建配置失败: " + err.Error())
		return
	}
	state = sharedCompat.Normalize(state)
	now := time.Now().UnixMilli()
	profileID := fmt.Sprintf("gio-%d", now)
	profile := sharedCompat.UpstreamProfile{
		ID:                 profileID,
		Name:               nextProfileName(state.Profiles),
		APIMode:            apiMode,
		RequestPolicy:      string(client.RequestPolicyOpenAI),
		ImagesNewAPICompat: false,
		TextModelID:        client.TextModel,
		ImageModelID:       client.ImageModel,
		CreatedAt:          now,
		LastUsedAt:         now,
	}
	state.Profiles = append(state.Profiles, profile)
	state.ActiveProfile = profileID
	state.UpdatedAt = now
	if err := gioCompat.SaveState(state); err != nil {
		a.appendLog("创建配置失败: " + err.Error())
		return
	}
	cfg := gioCompat.ConfigFromState(kernel.DefaultConfig(), state)
	a.applyRuntimeConfig(cfg)
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.activeProfileID = profileID
	a.status = "已创建配置: " + profile.Name
	a.logs = appendBounded(a.logs, "已创建配置: "+profile.Name)
	a.mu.Unlock()
	a.profileNameInput.SetText(profile.Name)
	a.concurrencyLimitInput.SetText("")
	a.invalidateNow()
}

func (a *App) duplicateActiveProfile() {
	a.saveCurrentConfig()
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("复制配置失败: " + err.Error())
		return
	}
	state = sharedCompat.Normalize(state)
	current, ok := currentActiveProfile(state)
	if !ok {
		return
	}
	now := time.Now().UnixMilli()
	profileID := fmt.Sprintf("gio-%d", now)
	clone := current
	clone.ID = profileID
	clone.Name = nextProfileName(state.Profiles)
	clone.CreatedAt = now
	clone.LastUsedAt = now
	state.Profiles = append(state.Profiles, clone)
	state.ActiveProfile = profileID
	state.UpdatedAt = now
	if key, _ := gioCompat.ReadAPIKey(current.ID); strings.TrimSpace(key) != "" {
		_ = gioCompat.WriteAPIKey(profileID, key)
	}
	if err := gioCompat.SaveState(state); err != nil {
		a.appendLog("复制配置失败: " + err.Error())
		return
	}
	cfg := gioCompat.ConfigFromState(kernel.DefaultConfig(), state)
	a.applyRuntimeConfig(cfg)
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.activeProfileID = profileID
	a.status = "已复制配置: " + clone.Name
	a.logs = appendBounded(a.logs, "已复制配置: "+clone.Name)
	a.mu.Unlock()
	a.profileNameInput.SetText(clone.Name)
	if clone.ConcurrencyLimit > 0 {
		a.concurrencyLimitInput.SetText(strconv.Itoa(clone.ConcurrencyLimit))
	} else {
		a.concurrencyLimitInput.SetText("")
	}
	a.invalidateNow()
}

func (a *App) deleteActiveProfile() {
	a.saveCurrentConfig()
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("删除配置失败: " + err.Error())
		return
	}
	state = sharedCompat.Normalize(state)
	if len(state.Profiles) == 0 {
		return
	}
	current, ok := currentActiveProfile(state)
	if !ok {
		return
	}
	nextProfiles := make([]sharedCompat.UpstreamProfile, 0, len(state.Profiles)-1)
	for _, profile := range state.Profiles {
		if profile.ID == current.ID {
			continue
		}
		nextProfiles = append(nextProfiles, profile)
	}
	state.Profiles = nextProfiles
	if len(nextProfiles) > 0 {
		state.ActiveProfile = nextProfiles[0].ID
	} else {
		state.ActiveProfile = ""
	}
	state.UpdatedAt = time.Now().UnixMilli()
	_ = gioCompat.WriteAPIKey(current.ID, "")
	if err := gioCompat.SaveState(state); err != nil {
		a.appendLog("删除配置失败: " + err.Error())
		return
	}
	cfg := gioCompat.ConfigFromState(kernel.DefaultConfig(), state)
	a.applyRuntimeConfig(cfg)
	a.mu.Lock()
	a.profiles = append([]sharedCompat.UpstreamProfile(nil), state.Profiles...)
	a.activeProfileID = state.ActiveProfile
	a.status = "已删除配置: " + current.Name
	a.logs = appendBounded(a.logs, "已删除配置: "+current.Name)
	a.mu.Unlock()
	a.profileNameInput.SetText(activeProfileName(state.Profiles, state.ActiveProfile))
	if limit := activeProfileConcurrencyLimit(state.Profiles, state.ActiveProfile); limit > 0 {
		a.concurrencyLimitInput.SetText(strconv.Itoa(limit))
	} else {
		a.concurrencyLimitInput.SetText("")
	}
	a.invalidateNow()
}

func currentActiveProfile(state sharedCompat.State) (sharedCompat.UpstreamProfile, bool) {
	if strings.TrimSpace(state.ActiveProfile) != "" {
		for _, profile := range state.Profiles {
			if profile.ID == state.ActiveProfile {
				return profile, true
			}
		}
	}
	if len(state.Profiles) == 0 {
		return sharedCompat.UpstreamProfile{}, false
	}
	return state.Profiles[0], true
}

func augmentPromptWithStyle(prompt string, styleTag string) string {
	prompt = strings.TrimSpace(prompt)
	suffix := strings.TrimSpace(styleSuffixes[strings.TrimSpace(styleTag)])
	if prompt == "" || suffix == "" {
		return prompt
	}
	return prompt + ", " + suffix
}

func buildPromptSuggestions(promptHistory []string, history []sharedCompat.HistoryItem) []string {
	out := make([]string, 0, 8)
	seen := map[string]struct{}{}
	push := func(text string) {
		text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
		if text == "" {
			return
		}
		if _, ok := seen[text]; ok {
			return
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	for _, text := range promptHistory {
		push(text)
		if len(out) >= 8 {
			return out
		}
	}
	for _, item := range history {
		push(item.Prompt)
		if len(out) >= 8 {
			return out
		}
	}
	return out
}

func (a *App) applyPromptSuggestion(text string) {
	current := strings.TrimSpace(a.promptInput.Text())
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if current == "" {
		a.promptInput.SetText(text)
	} else {
		a.promptInput.SetText(current + "\n" + text)
	}
	a.promptHelperOpen = false
	a.invalidateNow()
}

func (a *App) useResultPrompt(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	a.promptInput.SetText(text)
	a.appendLog("已应用为下次提示词")
	a.closeResultDetail()
	a.invalidateNow()
}

func (a *App) prefillControlsFromHistoryItem(item sharedCompat.HistoryItem) {
	if prompt := strings.TrimSpace(item.Prompt); prompt != "" {
		a.promptInput.SetText(prompt)
	}
	if negative := strings.TrimSpace(item.NegativePrompt); negative != "" {
		a.negativePromptInput.SetText(negative)
	}
	if mode := strings.TrimSpace(item.Mode); mode == string(client.ModeEdit) || mode == string(client.ModeGenerate) {
		a.mode = mode
	}
	if size := strings.TrimSpace(item.Size); size != "" {
		a.size = size
	}
	if quality := strings.TrimSpace(item.Quality); quality != "" {
		a.quality = quality
	}
	if outputFormat := strings.TrimSpace(item.OutputFormat); outputFormat != "" {
		a.format = outputFormat
	}
	if background := strings.TrimSpace(item.Background); background != "" {
		a.background = background
	}
	if item.OutputCompression != nil && *item.OutputCompression > 0 {
		a.outputCompressionInput.SetText(strconv.Itoa(*item.OutputCompression))
	}
	if fidelity := strings.TrimSpace(item.InputFidelity); fidelity != "" {
		a.inputFidelity = fidelity
	}
	if imageStyle := strings.TrimSpace(item.ImageStyle); imageStyle != "" {
		a.imageStyle = imageStyle
	}
	if moderation := strings.TrimSpace(item.Moderation); moderation != "" {
		a.moderation = moderation
	}
	if item.Seed != 0 {
		a.seedInput.SetText(strconv.FormatInt(item.Seed, 10))
	}
	if styleTag := strings.TrimSpace(item.StyleTag); styleTag != "" {
		a.styleTag = styleTag
	}
	if item.BatchIndex > 0 {
		a.batchCount = normalizeBatchCount(item.BatchIndex + 1)
	}
	if strings.TrimSpace(item.Mode) == string(client.ModeEdit) && strings.TrimSpace(item.SavedPath) != "" {
		a.sourcePathsInput.SetText(strings.TrimSpace(item.SavedPath))
	}
}

func (a *App) applyPreset(preset sharedCompat.Preset) {
	if strings.TrimSpace(preset.Size) != "" {
		a.size = preset.Size
	}
	if strings.TrimSpace(preset.Quality) != "" {
		a.quality = preset.Quality
	}
	if strings.TrimSpace(preset.OutputFormat) != "" {
		a.format = preset.OutputFormat
	}
	a.negativePromptInput.SetText(strings.TrimSpace(preset.NegativePrompt))
	a.batchCount = normalizeBatchCount(preset.BatchCount)
	a.promptHelperOpen = false
	a.invalidateNow()
}

func (a *App) rememberPrompt(text string) {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return
	}
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("更新提示词历史失败: " + err.Error())
		return
	}
	next := []string{text}
	for _, existing := range state.Settings.PromptHistory {
		normalized := strings.Join(strings.Fields(strings.TrimSpace(existing)), " ")
		if normalized == "" || normalized == text {
			continue
		}
		next = append(next, normalized)
		if len(next) >= 50 {
			break
		}
	}
	state.Settings.PromptHistory = next
	state.UpdatedAt = time.Now().UnixMilli()
	if err := gioCompat.SaveState(state); err != nil {
		a.appendLog("保存提示词历史失败: " + err.Error())
		return
	}
	a.mu.Lock()
	a.promptHistory = append([]string(nil), next...)
	a.presets = append([]sharedCompat.Preset(nil), state.Settings.Presets...)
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) sourcePaths() []string {
	return kernel.ParseSourcePaths(a.sourcePathsInput.Text())
}

func (a *App) setSourcePaths(paths []string) {
	a.sourcePathsInput.SetText(strings.Join(paths, "\n"))
	a.invalidateNow()
}

func (a *App) removeSourcePath(target string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	next := make([]string, 0, len(a.sourcePaths()))
	for _, path := range a.sourcePaths() {
		if strings.TrimSpace(path) == target {
			continue
		}
		next = append(next, path)
	}
	a.setSourcePaths(next)
}

func (a *App) appendSourcePath(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	paths := a.sourcePaths()
	for _, existing := range paths {
		if strings.TrimSpace(existing) == path {
			return
		}
	}
	paths = append(paths, path)
	a.setSourcePaths(paths)
}

func (a *App) reuseHistoryItemAsSource(item sharedCompat.HistoryItem) {
	if strings.TrimSpace(item.SavedPath) == "" {
		return
	}
	a.appendSourcePath(item.SavedPath)
}

func (a *App) deleteHistoryItem(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("删除历史失败: " + err.Error())
		return
	}
	next := make([]sharedCompat.HistoryItem, 0, len(state.History))
	for _, item := range state.History {
		if item.ID == id {
			continue
		}
		next = append(next, item)
	}
	state.History = next
	state.UpdatedAt = time.Now().UnixMilli()
	if err := gioCompat.SaveState(state); err != nil {
		a.appendLog("删除历史失败: " + err.Error())
		return
	}
	a.mu.Lock()
	a.history = append([]sharedCompat.HistoryItem(nil), next...)
	if len(a.batchResultIDs) > 0 {
		kept := make([]string, 0, len(a.batchResultIDs))
		for _, batchID := range a.batchResultIDs {
			if batchID != id {
				kept = append(kept, batchID)
			}
		}
		a.batchResultIDs = kept
		if !a.canOpenResultGridLocked() {
			a.resultGridOpen = false
		}
	}
	if a.selectedHistoryID == id {
		a.selectedHistoryID = ""
	}
	if a.compare.Item.ID == id {
		a.compare = resultState{Rev: a.compare.Rev + 1}
		a.compareSplitSlider.Value = 0.5
	}
	if a.activeResultDetail.ID == id {
		a.activeResultDetail = sharedCompat.HistoryItem{}
	}
	if a.result.Item.ID == id {
		a.result = resultState{Rev: a.result.Rev + 1}
	}
	if a.activePromptGroup.Key != "" && historyPromptGroupContains(a.activePromptGroup, id) {
		a.activePromptGroup = historyPromptGroup{}
	}
	a.logs = appendBounded(a.logs, "已删除历史项: "+id)
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) persistThemeMode(mode string) {
	state, _, err := gioCompat.LoadState()
	if err != nil {
		a.appendLog("保存主题失败: " + err.Error())
		return
	}
	state.Settings.Theme = normalizeThemeMode(mode)
	state.UpdatedAt = time.Now().UnixMilli()
	if err := gioCompat.SaveState(state); err != nil {
		a.appendLog("保存主题失败: " + err.Error())
		return
	}
	a.applyThemeMode(state.Settings.Theme)
}

func (a *App) openPromptGroup(group historyPromptGroup) {
	a.mu.Lock()
	a.activePromptGroup = group
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) closePromptGroup() {
	a.mu.Lock()
	a.activePromptGroup = historyPromptGroup{}
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) startPromptOptimize() {
	if a.isRunning() {
		return
	}
	cfg := a.currentConfig()
	a.mu.Lock()
	if a.optimizingPrompt {
		a.mu.Unlock()
		return
	}
	a.optimizingPrompt = true
	a.logs = appendBounded(a.logs, "开始优化提示词")
	a.mu.Unlock()
	a.invalidateNow()

	go func() {
		optimized, err := kernel.OptimizePrompt(context.Background(), cfg)
		a.mu.Lock()
		a.optimizingPrompt = false
		a.mu.Unlock()
		if err != nil {
			a.appendLog("优化提示词失败: " + err.Error())
			return
		}
		optimized = strings.TrimSpace(optimized)
		if optimized == "" {
			a.appendLog("优化提示词失败: 上游返回空结果")
			return
		}
		a.promptInput.SetText(optimized)
		a.rememberPrompt(optimized)
		a.appendLog("提示词已优化")
		a.invalidateNow()
	}()
}

func (a *App) startUpstreamProbe() {
	if a.isRunning() {
		return
	}
	cfg := a.currentConfig()
	a.mu.Lock()
	if a.testingUpstream {
		a.mu.Unlock()
		return
	}
	a.testingUpstream = true
	a.lastProbeSummary = ""
	a.logs = appendBounded(a.logs, "开始测试上游连接")
	a.mu.Unlock()
	a.invalidateNow()

	go func() {
		result, err := kernel.ProbeUpstream(context.Background(), cfg)
		a.mu.Lock()
		a.testingUpstream = false
		if err != nil {
			a.lastProbeSummary = "测试失败"
			a.mu.Unlock()
			a.appendLog("上游测试失败: " + err.Error())
			return
		}
		a.lastProbeSummary = fmt.Sprintf("已连接 · %d models", result.ModelCount)
		a.logs = appendBounded(a.logs, fmt.Sprintf("上游测试成功: 发现 %d 个 models", result.ModelCount))
		a.mu.Unlock()
		a.invalidateNow()
	}()
}

func (a *App) replaceCurrentResultWithPath(path string, sourceEvent string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("变换输出路径为空")
	}
	img, err := decodeImageFile(path)
	if err != nil {
		return err
	}
	a.mu.Lock()
	item := a.result.Item
	item.SavedPath = path
	if item.ID == "" {
		item.ID = "derived:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	a.result = resultState{
		Image:         img,
		SavedPath:     path,
		RawPath:       a.result.RawPath,
		RevisedPrompt: a.result.RevisedPrompt,
		SourceEvent:   sourceEvent,
		Item:          item,
		HasItem:       true,
		Rev:           a.result.Rev + 1,
	}
	a.compare = resultState{Rev: a.compare.Rev + 1}
	a.compareSplitSlider.Value = 0.5
	a.selectedHistoryID = ""
	a.mu.Unlock()
	a.invalidateNow()
	return nil
}

func (a *App) clearCurrentResult() {
	a.mu.Lock()
	a.result = resultState{Rev: a.result.Rev + 1}
	a.compare = resultState{Rev: a.compare.Rev + 1}
	a.compareSplitSlider.Value = 0.5
	a.selectedHistoryID = ""
	a.activePromptGroup = historyPromptGroup{}
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) isCompareItem(item sharedCompat.HistoryItem) bool {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.compare.Item.ID == id
}

func (a *App) toggleCompareItem(item sharedCompat.HistoryItem) error {
	if strings.TrimSpace(item.ID) == "" {
		a.clearCompare()
		return nil
	}
	if a.isCompareItem(item) {
		a.clearCompare()
		return nil
	}
	return a.setCompareItem(item)
}

func (a *App) setCompareItem(item sharedCompat.HistoryItem) error {
	if strings.TrimSpace(item.ID) == "" && strings.TrimSpace(item.SavedPath) == "" {
		a.clearCompare()
		return nil
	}
	img, err := a.imageForHistoryItem(item)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.compare = resultState{
		Image:         img,
		SavedPath:     item.SavedPath,
		RawPath:       item.RawPath,
		RevisedPrompt: item.RevisedPrompt,
		SourceEvent:   "compare",
		Item:          item,
		HasItem:       item.ID != "",
		Rev:           a.compare.Rev + 1,
	}
	a.compareSplitSlider.Value = 0.5
	a.mu.Unlock()
	a.invalidateNow()
	return nil
}

func (a *App) clearCompare() {
	a.mu.Lock()
	a.compare = resultState{Rev: a.compare.Rev + 1}
	a.compareSplitSlider.Value = 0.5
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) toggleFullscreen() {
	a.mu.Lock()
	next := !a.fullscreen
	a.fullscreen = next
	window := a.window
	a.mu.Unlock()
	if window != nil {
		if next {
			window.Option(app.Fullscreen.Option())
		} else {
			window.Option(app.Windowed.Option())
		}
	}
	a.invalidateNow()
}

func (a *App) openResultDetail(item sharedCompat.HistoryItem) {
	if strings.TrimSpace(item.ID) == "" && strings.TrimSpace(item.SavedPath) == "" {
		return
	}
	a.mu.Lock()
	a.activeResultDetail = item
	a.mu.Unlock()
	a.invalidateNow()
}

func (a *App) closeResultDetail() {
	a.mu.Lock()
	a.activeResultDetail = sharedCompat.HistoryItem{}
	a.mu.Unlock()
	a.invalidateNow()
}
