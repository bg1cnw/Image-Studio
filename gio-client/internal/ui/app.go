package ui

import (
	"context"
	"image"
	"strconv"
	"strings"
	"sync"
	"time"

	gioCompat "image-studio/gio-client/internal/compat"
	"image-studio/gio-client/internal/kernel"
	sharedCompat "image-studio/shared/compat"

	"gioui.org/app"
	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type resultState struct {
	Image         image.Image
	SavedPath     string
	RawPath       string
	RevisedPrompt string
	SourceEvent   string
	Item          sharedCompat.HistoryItem
	HasItem       bool
	Rev           int
}

type snapshot struct {
	Running                   bool
	Status                    string
	Logs                      []string
	History                   []sharedCompat.HistoryItem
	BatchResults              []sharedCompat.HistoryItem
	BatchTotal                int
	Profiles                  []sharedCompat.UpstreamProfile
	ActiveProfileID           string
	SettingsSelectedProfileID string
	SelectedHistoryID         string
	PromptHistory             []string
	Presets                   []sharedCompat.Preset
	OptimizingPrompt          bool
	TestingUpstream           bool
	SyncingCodexConfig        bool
	LastProbeSummary          string
	ActivePromptGroup         historyPromptGroup
	ActiveResultDetail        sharedCompat.HistoryItem
	HistoryTimelineOpen       bool
	Fullscreen                bool
	LastErrorMessage          string
	LastRunAvailable          bool
	RawResponseModalPath      string
	RawResponseModalText      string
	RawResponseModalError     string
	ResultGridOpen            bool
	Compare                   resultState
	CompareSplit              float32
	Result                    resultState
	SavePromptVisible         bool
}

type cachedImage struct {
	Image  image.Image
	Failed bool
}

type workspaceState struct {
	ID                  string
	Name                string
	Prompt              string
	NegativePrompt      string
	Mode                string
	Size                string
	Quality             string
	OutputFormat        string
	StyleTag            string
	SeedText            string
	BatchCount          int
	SourcePathsText     string
	ResultSavedPath     string
	ResultRawPath       string
	ResultRevisedPrompt string
	ResultSourceEvent   string
	ResultItem          sharedCompat.HistoryItem
	ResultHasItem       bool
	SelectedHistoryID   string
	BatchResultIDs      []string
	ResultGridOpen      bool
	CompareHistoryID    string
	CompareSplit        float32
}

type App struct {
	th     *material.Theme
	runner kernel.Runner

	controlsList        widget.List
	logList             widget.List
	historyList         widget.List
	historyTimelineList widget.List
	promptGroupList     widget.List
	promptHelperList    widget.List
	settingsProfileList widget.List
	settingsList        widget.List
	workspaceList       widget.List

	apiKeyInput               widget.Editor
	baseURLInput              widget.Editor
	textModelInput            widget.Editor
	imageModelInput           widget.Editor
	profileNameInput          widget.Editor
	concurrencyLimitInput     widget.Editor
	promptInput               widget.Editor
	sourcePathsInput          widget.Editor
	outputDirInput            widget.Editor
	seedInput                 widget.Editor
	negativePromptInput       widget.Editor
	partialImagesInput        widget.Editor
	proxyURLInput             widget.Editor
	savePromptPathInput       widget.Editor
	rawResponseViewerInput    widget.Editor
	historyQueryInput         widget.Editor
	historyTimelineQueryInput widget.Editor
	workspaceNameInput        widget.Editor

	mode       string
	api        string
	size       string
	quality    string
	format     string
	policy     string
	proxy      string
	styleTag   string
	themeMode  string
	fontScale  float64
	batchCount int

	modeButtons                      []widget.Clickable
	apiButtons                       []widget.Clickable
	sizeButtons                      []widget.Clickable
	aspectButtons                    []widget.Clickable
	styleButtons                     []widget.Clickable
	clearStyleButton                 widget.Clickable
	randomSeedButton                 widget.Clickable
	clearSeedButton                  widget.Clickable
	batchCountButtons                []widget.Clickable
	resolutionButtons                []widget.Clickable
	qualityButtons                   []widget.Clickable
	formatButtons                    []widget.Clickable
	policyButtons                    []widget.Clickable
	proxyButtons                     []widget.Clickable
	historyModeButtons               []widget.Clickable
	historyDateButtons               []widget.Clickable
	historyTimelineModeButtons       []widget.Clickable
	historyTimelineDateButtons       []widget.Clickable
	runButton                        widget.Clickable
	cancelButton                     widget.Clickable
	retryLastRunButton               widget.Clickable
	openRawResponseButton            widget.Clickable
	openLogsRawResponseButton        widget.Clickable
	dismissErrorButton               widget.Clickable
	closeRawResponseButton           widget.Clickable
	copyRawResponseButton            widget.Clickable
	clearLogButton                   widget.Clickable
	saveAsButton                     widget.Clickable
	latestResultButton               widget.Clickable
	currentGroupButton               widget.Clickable
	closeCompareButton               widget.Clickable
	closeResultGridButton            widget.Clickable
	rotateLeftButton                 widget.Clickable
	rotateRightButton                widget.Clickable
	flipHorizontalButton             widget.Clickable
	flipVerticalButton               widget.Clickable
	clearCurrentButton               widget.Clickable
	clearSourcesButton               widget.Clickable
	addSourceFilesButton             widget.Clickable
	addSourceStripButton             widget.Clickable
	emptyStateImportButton           widget.Clickable
	promptHelperButton               widget.Clickable
	promptHelperTemplatesButton      widget.Clickable
	promptHelperHistoryButton        widget.Clickable
	closePromptHelperButton          widget.Clickable
	optimizePromptButton             widget.Clickable
	testUpstreamButton               widget.Clickable
	settingsTestUpstreamButton       widget.Clickable
	syncCodexConfigButton            widget.Clickable
	historyTimelineModePickerButton  widget.Clickable
	historyTimelineDatePickerButton  widget.Clickable
	toggleAPIKeyMaskButton           widget.Clickable
	upstreamConfigButton             widget.Clickable
	settingsHelpButton               widget.Clickable
	closeSettingsHelpButton          widget.Clickable
	saveSettingsButton               widget.Clickable
	closeGeneralSettingsButton       widget.Clickable
	generalRuntimePickerButton       widget.Clickable
	openGeneralUpstreamButton        widget.Clickable
	openGeneralOutputButton          widget.Clickable
	chooseGeneralOutputButton        widget.Clickable
	resetGeneralOutputButton         widget.Clickable
	openGeneralHistoryTimelineButton widget.Clickable
	exportGeneralHistoryButton       widget.Clickable
	importGeneralHistoryButton       widget.Clickable
	openGeneralAboutButton           widget.Clickable
	clearGeneralAPIKeyButton         widget.Clickable
	clearGeneralHistoryButton        widget.Clickable
	pruneGeneralHistoryButtons       []widget.Clickable
	openGeneralRepoButton            widget.Clickable
	openGeneralFeedbackButton        widget.Clickable
	closeAboutButton                 widget.Clickable
	openAboutRepoButton              widget.Clickable
	openAboutFeedbackButton          widget.Clickable
	openAboutLicenseButton           widget.Clickable
	themeButtons                     []widget.Clickable
	generalThemeButtons              []widget.Clickable
	generalRuntimeButtons            []widget.Clickable
	generalFontScaleButtons          []widget.Clickable
	generalSavePromptButtons         []widget.Clickable
	generalProxyButtons              []widget.Clickable
	generalKeepLogsButtons           []widget.Clickable
	headerAddWorkspaceButton         widget.Clickable
	headerQuoteButton                widget.Clickable
	githubButton                     widget.Clickable
	headerStarButton                 widget.Clickable
	settingsButton                   widget.Clickable
	fullscreenButton                 widget.Clickable
	resultDetailButton               widget.Clickable
	footerOutputButton               widget.Clickable
	footerGithubButton               widget.Clickable
	footerFeedbackButton             widget.Clickable
	addWorkspaceButton               widget.Clickable
	workspaceRenameSaveButton        widget.Clickable
	workspaceRenameCancelButton      widget.Clickable
	closeSettingsButton              widget.Clickable
	createProfileButton              widget.Clickable
	createImagesProfileButton        widget.Clickable
	duplicateProfileButton           widget.Clickable
	deleteProfileButton              widget.Clickable
	settingsActivateProfileButton    widget.Clickable
	closeResultDetailButton          widget.Clickable
	resultDetailSaveAsButton         widget.Clickable
	resultDetailUseSourceButton      widget.Clickable
	resultDetailUsePromptButton      widget.Clickable
	resultDetailUseRevisedButton     widget.Clickable
	resultDetailOpenPathButton       widget.Clickable
	resultDetailCopyPromptButton     widget.Clickable
	resultDetailCopyRevisedButton    widget.Clickable
	resultDetailCopyPathButton       widget.Clickable
	resultDetailDeleteButton         widget.Clickable
	composeToggleButton              widget.Clickable
	advancedToggleButton             widget.Clickable
	profilePickerButton              widget.Clickable
	manageUpstreamButton             widget.Clickable
	historyCollapseButton            widget.Clickable
	closePromptGroupButton           widget.Clickable
	openHistoryTimelineButton        widget.Clickable
	openHistoryTimelineMoreButton    widget.Clickable
	closeHistoryTimelineButton       widget.Clickable
	savePromptSaveButton             widget.Clickable
	savePromptSkipButton             widget.Clickable
	savePromptNeverAsk               widget.Bool

	mu                    sync.Mutex
	running               bool
	cancel                context.CancelFunc
	status                string
	logs                  []string
	history               []sharedCompat.HistoryItem
	profiles              []sharedCompat.UpstreamProfile
	promptHistory         []string
	presets               []sharedCompat.Preset
	activeProfileID       string
	selectedHistoryID     string
	optimizingPrompt      bool
	testingUpstream       bool
	syncingCodexConfig    bool
	lastProbeSummary      string
	fullscreen            bool
	activeResultDetail    sharedCompat.HistoryItem
	result                resultState
	compare               resultState
	imageOp               paint.ImageOp
	imageOpRev            int
	imageCache            map[string]cachedImage
	lastRunConfig         kernel.Config
	lastRunBatchCount     int
	lastRunValid          bool
	lastErrorMessage      string
	rawResponseModalPath  string
	rawResponseModalText  string
	rawResponseModalError string
	batchResultIDs        []string
	resultGridOpen        bool
	compareSplitSlider    widget.Float
	compareSplitDrag      gesture.Drag

	savePromptVisible             bool
	savePromptSuppressed          bool
	keepLogs                      bool
	kernelRuntimeMode             string
	savePromptSourcePath          string
	composeOpen                   bool
	advancedOpen                  bool
	profilePickerOpen             bool
	historyRailCollapsed          bool
	historyModeFilter             string
	historyDateFilter             string
	historyTimelineOpen           bool
	historyTimelineModeFilter     string
	historyTimelineDateFilter     string
	historyTimelineModePickerOpen bool
	historyTimelineDatePickerOpen bool
	profileButtons                map[string]*widget.Clickable
	settingsProfileButtons        map[string]*widget.Clickable
	historyButtons                map[string]*widget.Clickable
	promptButtons                 map[string]*widget.Clickable
	sourceButtons                 map[string]*widget.Clickable
	historyActionButtons          map[string]*widget.Clickable
	expandedPromptGroups          map[string]bool
	promptHelperOpen              bool
	promptHelperTab               string
	activePromptGroup             historyPromptGroup
	generalSettingsOpen           bool
	generalRuntimePickerOpen      bool
	aboutModalOpen                bool
	settingsModalOpen             bool
	settingsHelpOpen              bool
	settingsSelectedProfileID     string
	apiKeyVisible                 bool
	workspaces                    []workspaceState
	activeWorkspaceID             string
	workspaceButtons              map[string]*widget.Clickable
	closeWorkspaceButtons         map[string]*widget.Clickable
	workspaceRenameID             string
	workspaceLastClickID          string
	workspaceLastClickAt          time.Time
	headerQuoteIndex              int

	invalidate func()
	window     *app.Window
}

func New() *App {
	cfg := kernel.DefaultConfig()
	compatState, compatPath, compatErr := gioCompat.LoadState()
	if compatErr == nil {
		cfg = gioCompat.ConfigFromState(cfg, compatState)
	}
	themeMode := normalizeThemeMode(compatState.Settings.Theme)
	fontScale := normalizeFontScale(compatState.Settings.FontScale)
	fluent = themePalette(resolveThemeMode(themeMode))
	th := material.NewTheme()
	collection := bundledFontCollection()
	if len(collection) > 0 {
		th.Shaper = text.NewShaper(text.WithCollection(collection))
	} else {
		th.Shaper = text.NewShaper()
	}
	th.Face = uiSansTypeface
	th.Palette = material.Palette{
		Bg:         fluent.bg,
		Fg:         fluent.text,
		ContrastBg: fluent.accent,
		ContrastFg: fluent.white,
	}
	th.TextSize = unit.Sp(float32(14) * float32(fontScale))
	a := &App{
		th:                         th,
		runner:                     kernel.Runner{},
		mode:                       string(cfg.Mode),
		api:                        string(cfg.APIMode),
		size:                       cfg.Size,
		quality:                    cfg.Quality,
		format:                     cfg.OutputFormat,
		policy:                     string(cfg.RequestPolicy),
		proxy:                      cfg.ProxyMode,
		styleTag:                   "",
		themeMode:                  themeMode,
		fontScale:                  fontScale,
		kernelRuntimeMode:          normalizeKernelRuntimeMode(compatState.Settings.KernelRuntimeMode),
		batchCount:                 1,
		themeButtons:               make([]widget.Clickable, 3),
		generalThemeButtons:        make([]widget.Clickable, 3),
		generalRuntimeButtons:      make([]widget.Clickable, 3),
		generalFontScaleButtons:    make([]widget.Clickable, 3),
		generalSavePromptButtons:   make([]widget.Clickable, 2),
		generalProxyButtons:        make([]widget.Clickable, len(proxyChoices)),
		generalKeepLogsButtons:     make([]widget.Clickable, 2),
		pruneGeneralHistoryButtons: make([]widget.Clickable, 2),
		modeButtons:                make([]widget.Clickable, len(modeChoices)),
		apiButtons:                 make([]widget.Clickable, len(apiChoices)),
		sizeButtons:                make([]widget.Clickable, len(sizeChoices)),
		aspectButtons:              make([]widget.Clickable, len(aspectChoices)),
		styleButtons:               make([]widget.Clickable, len(styleChoices)),
		batchCountButtons:          make([]widget.Clickable, len(batchCountChoices)),
		resolutionButtons:          make([]widget.Clickable, len(resolutionChoices)),
		qualityButtons:             make([]widget.Clickable, len(qualityChoices)),
		formatButtons:              make([]widget.Clickable, len(formatChoices)),
		policyButtons:              make([]widget.Clickable, len(policyChoices)),
		proxyButtons:               make([]widget.Clickable, len(proxyChoices)),
		historyModeButtons:         make([]widget.Clickable, 3),
		historyDateButtons:         make([]widget.Clickable, 3),
		historyTimelineModeButtons: make([]widget.Clickable, 3),
		historyTimelineDateButtons: make([]widget.Clickable, 3),
		status:                     "Gio 原生客户端就绪",
		logs:                       []string{"独立 Gio 高性能测试客户端已启动。"},
		history:                    append([]sharedCompat.HistoryItem(nil), compatState.History...),
		profiles:                   append([]sharedCompat.UpstreamProfile(nil), compatState.Profiles...),
		promptHistory:              append([]string(nil), compatState.Settings.PromptHistory...),
		presets:                    append([]sharedCompat.Preset(nil), compatState.Settings.Presets...),
		savePromptSuppressed:       gioCompat.SavePromptSuppressed(compatState),
		keepLogs:                   compatState.Settings.KeepLogs,
		imageCache:                 map[string]cachedImage{},
		composeOpen:                false,
		advancedOpen:               false,
		profilePickerOpen:          false,
		historyRailCollapsed:       false,
		historyModeFilter:          "all",
		historyDateFilter:          "all",
		historyTimelineModeFilter:  "all",
		historyTimelineDateFilter:  "all",
		profileButtons:             map[string]*widget.Clickable{},
		settingsProfileButtons:     map[string]*widget.Clickable{},
		historyButtons:             map[string]*widget.Clickable{},
		promptButtons:              map[string]*widget.Clickable{},
		sourceButtons:              map[string]*widget.Clickable{},
		historyActionButtons:       map[string]*widget.Clickable{},
		workspaceButtons:           map[string]*widget.Clickable{},
		closeWorkspaceButtons:      map[string]*widget.Clickable{},
		expandedPromptGroups:       map[string]bool{},
		promptHelperOpen:           false,
		promptHelperTab:            "templates",
		headerQuoteIndex:           initialHeaderQuoteIndex(time.Now()),
	}
	if profile, ok := gioCompat.ActiveProfile(compatState); ok {
		a.activeProfileID = profile.ID
		a.profileNameInput.SetText(strings.TrimSpace(profile.Name))
		if profile.ConcurrencyLimit > 0 {
			a.concurrencyLimitInput.SetText(strconv.Itoa(profile.ConcurrencyLimit))
		}
	}
	a.savePromptNeverAsk.Value = a.savePromptSuppressed
	if compatPath != "" {
		a.logs = append(a.logs, "兼容状态文件: "+compatPath)
	}
	if compatErr != nil {
		a.logs = append(a.logs, "兼容状态读取失败: "+compatErr.Error())
	}
	a.controlsList.List.Axis = layout.Vertical
	a.logList.List.Axis = layout.Vertical
	a.historyList.List.Axis = layout.Vertical
	a.historyTimelineList.List.Axis = layout.Vertical
	a.promptGroupList.List.Axis = layout.Vertical
	a.promptHelperList.List.Axis = layout.Vertical
	a.settingsProfileList.List.Axis = layout.Vertical
	a.settingsList.List.Axis = layout.Vertical
	a.workspaceList.List.Axis = layout.Horizontal
	a.compareSplitSlider.Value = 0.5
	a.configureEditors(cfg)
	a.historyQueryInput.SingleLine = true
	a.historyTimelineQueryInput.SingleLine = true
	if latest, ok := newestHistoryItem(a.history); ok {
		if err := a.loadHistoryPreview(latest, false); err != nil && !isMissingPreview(err) {
			a.logs = appendBounded(a.logs, "载入最近历史失败: "+err.Error())
		}
	}
	a.initWorkspaces()
	return a
}

func (a *App) configureEditors(cfg kernel.Config) {
	singleLine := []*widget.Editor{
		&a.apiKeyInput,
		&a.baseURLInput,
		&a.textModelInput,
		&a.imageModelInput,
		&a.profileNameInput,
		&a.concurrencyLimitInput,
		&a.outputDirInput,
		&a.seedInput,
		&a.partialImagesInput,
		&a.proxyURLInput,
		&a.savePromptPathInput,
		&a.historyQueryInput,
		&a.historyTimelineQueryInput,
		&a.workspaceNameInput,
	}
	for _, editor := range singleLine {
		editor.SingleLine = true
	}
	a.apiKeyInput.Mask = '*'
	a.workspaceNameInput.Submit = true
	a.rawResponseViewerInput.ReadOnly = true
	a.seedInput.Filter = "0123456789"
	a.partialImagesInput.Filter = "0123456789"
	a.concurrencyLimitInput.Filter = "0123456789"
	a.apiKeyInput.SetText(cfg.APIKey)
	a.baseURLInput.SetText(cfg.BaseURL)
	a.textModelInput.SetText(cfg.TextModelID)
	a.imageModelInput.SetText(cfg.ImageModelID)
	a.outputDirInput.SetText(cfg.OutputDir)
	a.partialImagesInput.SetText(strconv.Itoa(cfg.PartialImages))
	a.proxyURLInput.SetText(cfg.ProxyURL)
	a.promptInput.SetText("")
}

func (a *App) applyRuntimeConfig(cfg kernel.Config) {
	if strings.TrimSpace(cfg.APIKey) != "" || strings.TrimSpace(a.apiKeyInput.Text()) != "" {
		a.apiKeyInput.SetText(cfg.APIKey)
	}
	a.baseURLInput.SetText(cfg.BaseURL)
	a.textModelInput.SetText(cfg.TextModelID)
	a.imageModelInput.SetText(cfg.ImageModelID)
	a.proxy = cfg.ProxyMode
	a.proxyURLInput.SetText(cfg.ProxyURL)
	a.outputDirInput.SetText(cfg.OutputDir)
	if strings.TrimSpace(cfg.OutputFormat) != "" {
		a.format = cfg.OutputFormat
	}
	if strings.TrimSpace(string(cfg.APIMode)) != "" {
		a.api = string(cfg.APIMode)
	}
	if strings.TrimSpace(string(cfg.RequestPolicy)) != "" {
		a.policy = string(cfg.RequestPolicy)
	}
}

func (a *App) Run(w *app.Window) error {
	a.window = w
	a.invalidate = w.Invalidate
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			a.saveCurrentConfig()
			a.cancelRun()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			a.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}
