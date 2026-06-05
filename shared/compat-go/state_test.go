package compat

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsEmptyState(t *testing.T) {
	state, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if state.SchemaVersion != SchemaVersion {
		t.Fatalf("schema=%d want %d", state.SchemaVersion, SchemaVersion)
	}
	if state.UpdatedAt != 0 {
		t.Fatalf("updatedAt=%d want 0", state.UpdatedAt)
	}
	if state.Profiles == nil || state.History == nil {
		t.Fatalf("expected initialized slices: %#v", state)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := StatePath(t.TempDir())
	input := State{
		Client:    "test",
		UpdatedAt: 123,
		Settings: Settings{
			ProxyMode:            "custom",
			ProxyURL:             "http://127.0.0.1:7890",
			Theme:                "dark",
			OutputFormat:         "webp",
			OutputDir:            "/tmp/images",
			PromptHistory:        []string{"cat"},
			ReducedEffects:       true,
			SavePromptSuppressed: true,
			KeepLogs:             true,
			CompletionSound: &CompletionSoundSettings{
				Enabled:    true,
				Mode:       "custom",
				CustomName: "ding.wav",
				CustomData: "data:audio/wav;base64,AAAA",
			},
		},
		Profiles: []UpstreamProfile{{
			ID:            "p1",
			Name:          "配置1",
			APIMode:       "responses",
			RequestPolicy: "openai",
			BaseURL:       "https://upstream.example",
			TextModelID:   "gpt-5.5",
			ImageModelID:  "gpt-image-2",
			CreatedAt:     100,
		}},
		ActiveProfile: "p1",
		History: []HistoryItem{{
			ID:           "h1",
			Prompt:       "cat",
			Mode:         "generate",
			Size:         "1024x1024",
			Quality:      "high",
			OutputFormat: "png",
			CreatedAt:    200,
			SourcePaths:  []string{"/tmp/a.png", "/tmp/b.png"},
			SavedPath:    "/tmp/images/cat.png",
		}},
		HistoryFull: []HistoryFullItem{{ID: "h1", ImageB64: "aW1n"}},
	}
	if err := Save(path, input); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Client != "test" || loaded.ActiveProfile != "p1" || loaded.Settings.OutputDir != "/tmp/images" || !loaded.Settings.ReducedEffects || !loaded.Settings.SavePromptSuppressed || !loaded.Settings.KeepLogs {
		t.Fatalf("unexpected state: %#v", loaded)
	}
	if loaded.Settings.CompletionSound == nil || !loaded.Settings.CompletionSound.Enabled || loaded.Settings.CompletionSound.Mode != "custom" || loaded.Settings.CompletionSound.CustomName != "ding.wav" || loaded.Settings.CompletionSound.CustomData != "data:audio/wav;base64,AAAA" {
		t.Fatalf("completion sound not preserved: %#v", loaded.Settings.CompletionSound)
	}
	if len(loaded.Profiles) != 1 || loaded.Profiles[0].BaseURL != "https://upstream.example" {
		t.Fatalf("profiles not preserved: %#v", loaded.Profiles)
	}
	if len(loaded.History) != 1 || loaded.History[0].SavedPath != "/tmp/images/cat.png" {
		t.Fatalf("history not preserved: %#v", loaded.History)
	}
	if len(loaded.History[0].SourcePaths) != 2 || loaded.History[0].SourcePaths[0] != "/tmp/a.png" || loaded.History[0].SourcePaths[1] != "/tmp/b.png" {
		t.Fatalf("sourcePaths not preserved: %#v", loaded.History[0].SourcePaths)
	}
	if len(loaded.HistoryFull) != 1 || loaded.HistoryFull[0].ImageB64 != "aW1n" {
		t.Fatalf("historyFull not preserved: %#v", loaded.HistoryFull)
	}
}

func TestLoadInvalidJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := Load(path); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected parse error, got %v", err)
	}
}
