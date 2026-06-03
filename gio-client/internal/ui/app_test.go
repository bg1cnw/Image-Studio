package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sharedCompat "image-studio/shared/compat"
)

func TestCopyImageFileCopiesToExplicitPath(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.png")
	if err := os.WriteFile(src, []byte("image"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	dst := filepath.Join(dir, "nested", "copy.png")
	saved, err := copyImageFile(src, dst)
	if err != nil {
		t.Fatalf("copyImageFile: %v", err)
	}
	if saved != dst {
		t.Fatalf("saved=%q want %q", saved, dst)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read copied: %v", err)
	}
	if string(data) != "image" {
		t.Fatalf("copied data=%q", data)
	}
}

func TestCopyImageFileDirectoryTargetKeepsSourceName(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.webp")
	if err := os.WriteFile(src, []byte("image"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	targetDir := filepath.Join(dir, "target")
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	saved, err := copyImageFile(src, targetDir)
	if err != nil {
		t.Fatalf("copyImageFile: %v", err)
	}
	want := filepath.Join(targetDir, "source.webp")
	if saved != want {
		t.Fatalf("saved=%q want %q", saved, want)
	}
}

func TestMatchHistoryQueryMatchesPromptAndPath(t *testing.T) {
	item := sharedCompat.HistoryItem{
		Prompt:        "生成一张雪山海报",
		RevisedPrompt: "cinematic snow mountain poster",
		SavedPath:     "/tmp/snow.png",
		Size:          "1024x1024",
		Quality:       "high",
	}
	if !matchHistoryQuery(item, "雪山") {
		t.Fatalf("expected prompt match")
	}
	if !matchHistoryQuery(item, "snow.png") {
		t.Fatalf("expected path match")
	}
	if matchHistoryQuery(item, "desert") {
		t.Fatalf("unexpected query match")
	}
}

func TestTodayHistoryCountUsesLocalDayBoundary(t *testing.T) {
	now := time.Date(2026, time.May, 31, 15, 4, 0, 0, time.Local)
	items := []sharedCompat.HistoryItem{
		{ID: "a", CreatedAt: now.Add(-2 * time.Hour).UnixMilli()},
		{ID: "b", CreatedAt: now.Add(-26 * time.Hour).UnixMilli()},
	}
	if got := todayHistoryCount(items, now); got != 1 {
		t.Fatalf("todayHistoryCount=%d want 1", got)
	}
}

func TestFilteredHistoryItemsRespectsQueryModeAndDate(t *testing.T) {
	now := time.Date(2026, time.May, 31, 15, 4, 0, 0, time.Local)
	items := []sharedCompat.HistoryItem{
		{ID: "a", Prompt: "城市夜景", Mode: "generate", CreatedAt: now.Add(-2 * time.Hour).UnixMilli()},
		{ID: "b", Prompt: "城市夜景", Mode: "edit", CreatedAt: now.Add(-48 * time.Hour).UnixMilli()},
		{ID: "c", Prompt: "森林雾气", Mode: "generate", CreatedAt: now.Add(-2 * time.Hour).UnixMilli()},
	}
	got := filteredHistoryItems(items, "城市", "generate", "today", now)
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("filteredHistoryItems=%v want only a", got)
	}
}

func TestAugmentPromptWithStyle(t *testing.T) {
	got := augmentPromptWithStyle("一只猫坐在窗边", "anime")
	want := "一只猫坐在窗边, anime style, cel shading, vibrant colors, detailed illustration"
	if got != want {
		t.Fatalf("augmentPromptWithStyle=%q want %q", got, want)
	}
	if same := augmentPromptWithStyle("一只猫", ""); same != "一只猫" {
		t.Fatalf("augmentPromptWithStyle without style=%q", same)
	}
}

func TestNormalizeBatchCount(t *testing.T) {
	if got := normalizeBatchCount(0); got != 1 {
		t.Fatalf("normalizeBatchCount(0)=%d want 1", got)
	}
	if got := normalizeBatchCount(12); got != 9 {
		t.Fatalf("normalizeBatchCount(12)=%d want 9", got)
	}
	if got := normalizeBatchCount(4); got != 4 {
		t.Fatalf("normalizeBatchCount(4)=%d want 4", got)
	}
}

func TestBuildPromptSuggestionsMergesHistorySources(t *testing.T) {
	promptHistory := []string{"一只猫坐在窗边", "夜色城市海报"}
	history := []sharedCompat.HistoryItem{
		{ID: "a", Prompt: "夜色城市海报"},
		{ID: "b", Prompt: "山谷晨雾风景"},
	}
	got := buildPromptSuggestions(promptHistory, history)
	if len(got) != 3 {
		t.Fatalf("len(buildPromptSuggestions)=%d want 3", len(got))
	}
	if got[0] != "一只猫坐在窗边" || got[2] != "山谷晨雾风景" {
		t.Fatalf("buildPromptSuggestions=%v", got)
	}
}

func TestFindPromptGroupForItemReturnsGroupedItems(t *testing.T) {
	items := []sharedCompat.HistoryItem{
		{ID: "1", Prompt: "cat poster"},
		{ID: "2", Prompt: "cat poster"},
		{ID: "3", Prompt: "dog poster"},
	}
	group, ok := findPromptGroupForItem(items, "2")
	if !ok {
		t.Fatalf("expected prompt group")
	}
	if len(group.Items) != 2 {
		t.Fatalf("group size=%d want 2", len(group.Items))
	}
}

func TestBuildHistoryDayGroupsKeepsPromptGroupsByDay(t *testing.T) {
	now := time.Date(2026, time.May, 31, 15, 4, 0, 0, time.Local)
	items := []sharedCompat.HistoryItem{
		{ID: "1", Prompt: "cat poster", CreatedAt: now.UnixMilli()},
		{ID: "2", Prompt: "cat poster", CreatedAt: now.Add(-2 * time.Hour).UnixMilli()},
		{ID: "3", Prompt: "dog poster", CreatedAt: now.Add(-26 * time.Hour).UnixMilli()},
	}
	groups := buildHistoryDayGroups(items)
	if len(groups) != 2 {
		t.Fatalf("len(buildHistoryDayGroups)=%d want 2", len(groups))
	}
	if groups[0].Label != "2026-05-31" || len(groups[0].Entries) != 1 || groups[0].Entries[0].Kind != "group" {
		t.Fatalf("unexpected first day group: %#v", groups[0])
	}
	if groups[1].Label != "2026-05-30" || len(groups[1].Entries) != 1 || groups[1].Entries[0].Item.ID != "3" {
		t.Fatalf("unexpected second day group: %#v", groups[1])
	}
}

func TestNextProfileNameFindsSmallestMissingNumber(t *testing.T) {
	profiles := []sharedCompat.UpstreamProfile{
		{Name: "配置1"},
		{Name: "配置3"},
	}
	if got := nextProfileName(profiles); got != "配置2" {
		t.Fatalf("nextProfileName=%q want 配置2", got)
	}
}

func TestWorkspaceSwitchPreservesPrompt(t *testing.T) {
	app := New()
	app.promptInput.SetText("workspace one")
	app.createWorkspace()
	if len(app.workspaces) != 2 {
		t.Fatalf("workspaces=%d want 2", len(app.workspaces))
	}
	second := app.activeWorkspaceID
	app.promptInput.SetText("workspace two")
	first := app.workspaces[0].ID
	app.switchWorkspace(first)
	if got := strings.TrimSpace(app.promptInput.Text()); got != "workspace one" {
		t.Fatalf("after switch back prompt=%q want workspace one", got)
	}
	app.switchWorkspace(second)
	if got := strings.TrimSpace(app.promptInput.Text()); got != "workspace two" {
		t.Fatalf("after switch second prompt=%q want workspace two", got)
	}
}

func TestWorkspaceRenameUpdatesState(t *testing.T) {
	app := New()
	id := app.activeWorkspaceID
	app.startWorkspaceRename(id)
	app.workspaceNameInput.SetText("封面方案")
	app.commitWorkspaceRename()
	if app.workspaces[0].Name != "封面方案" {
		t.Fatalf("workspace name=%q want 封面方案", app.workspaces[0].Name)
	}
}

func TestDisplayedWorkspaceNameUsesPromptForDefaultActiveWorkspace(t *testing.T) {
	app := New()
	app.promptInput.SetText("夜色城市概念海报")
	app.workspaces[0].Name = "图片 1"
	name := app.displayedWorkspaceName(app.workspaces[0])
	if name != "夜色城市概念海报" {
		t.Fatalf("displayedWorkspaceName=%q want 夜色城市概念海报", name)
	}
}

func TestResolveThemeMode(t *testing.T) {
	prev := systemThemeResolver
	systemThemeResolver = func() string { return "dark" }
	defer func() { systemThemeResolver = prev }()
	if got := resolveThemeMode("dark"); got != "dark" {
		t.Fatalf("resolveThemeMode(dark)=%q", got)
	}
	if got := resolveThemeMode("system"); got != "dark" {
		t.Fatalf("resolveThemeMode(system)=%q", got)
	}
	if got := normalizeThemeMode("unknown"); got != "system" {
		t.Fatalf("normalizeThemeMode(unknown)=%q", got)
	}
}

func TestToggleFullscreenWithoutWindow(t *testing.T) {
	app := &App{}
	app.toggleFullscreen()
	if !app.fullscreen {
		t.Fatalf("fullscreen should be true after first toggle")
	}
	app.toggleFullscreen()
	if app.fullscreen {
		t.Fatalf("fullscreen should be false after second toggle")
	}
}

func TestParseDialogPathsDeduplicatesAndTrims(t *testing.T) {
	got := parseDialogPaths(" /tmp/a.png \n\"/tmp/b.jpg\"\n/tmp/a.png\n")
	if len(got) != 2 {
		t.Fatalf("len(parseDialogPaths)=%d want 2", len(got))
	}
	if got[0] != "/tmp/a.png" || got[1] != "/tmp/b.jpg" {
		t.Fatalf("parseDialogPaths=%v", got)
	}
}
