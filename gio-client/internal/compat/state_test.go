package compat

import (
	"testing"

	"image-studio/gio-client/internal/kernel"
	shared "image-studio/shared/compat"

	"github.com/yuanhua/image-gptcodex/pkg/client"
)

func TestConfigFromStateUsesActiveProfileAndSettings(t *testing.T) {
	state := shared.State{
		Settings: shared.Settings{
			OutputDir:    "/tmp/out",
			OutputFormat: "webp",
			ProxyMode:    client.ProxyModeCustom,
			ProxyURL:     "http://127.0.0.1:7890",
		},
		Profiles: []shared.UpstreamProfile{
			{ID: "p1", Name: "配置1", BaseURL: "https://old.example", TextModelID: "old-text", ImageModelID: "old-image", APIMode: string(client.APIModeImages)},
			{ID: "p2", Name: "配置2", BaseURL: "https://new.example", TextModelID: "new-text", ImageModelID: "new-image", APIMode: string(client.APIModeResponses), RequestPolicy: string(client.RequestPolicyCompat)},
		},
		ActiveProfile: "p2",
	}
	cfg := ConfigFromState(kernel.DefaultConfig(), state)
	if cfg.OutputDir != "/tmp/out" || cfg.OutputFormat != "webp" {
		t.Fatalf("settings not applied: %#v", cfg)
	}
	if cfg.BaseURL != "https://new.example" || cfg.TextModelID != "new-text" || cfg.ImageModelID != "new-image" {
		t.Fatalf("profile not applied: %#v", cfg)
	}
	if cfg.APIMode != client.APIModeResponses || cfg.RequestPolicy != client.RequestPolicyCompat {
		t.Fatalf("api settings not applied: %#v", cfg)
	}
	if cfg.ProxyMode != client.ProxyModeCustom || cfg.ProxyURL != "http://127.0.0.1:7890" {
		t.Fatalf("proxy not applied: %#v", cfg)
	}
}

func TestUpsertConfigPreservesActiveProfileIdentity(t *testing.T) {
	state := shared.State{
		Profiles: []shared.UpstreamProfile{{
			ID:               "p1",
			Name:             "主配置",
			APIMode:          string(client.APIModeImages),
			RequestPolicy:    string(client.RequestPolicyOpenAI),
			BaseURL:          "https://old.example",
			TextModelID:      "old-text",
			ImageModelID:     "old-image",
			ConcurrencyLimit: 3,
			CreatedAt:        100,
		}},
		ActiveProfile: "p1",
	}
	cfg := kernel.Config{
		BaseURL:       "https://new.example",
		TextModelID:   "new-text",
		ImageModelID:  "new-image",
		APIMode:       client.APIModeResponses,
		RequestPolicy: client.RequestPolicyCompat,
		OutputFormat:  "jpeg",
		OutputDir:     "/tmp/images",
		ProxyMode:     client.ProxyModeNone,
	}
	next := UpsertConfig(state, cfg)
	if next.ActiveProfile != "p1" || len(next.Profiles) != 1 {
		t.Fatalf("unexpected profiles: %#v", next.Profiles)
	}
	profile := next.Profiles[0]
	if profile.Name != "主配置" || profile.CreatedAt != 100 || profile.ConcurrencyLimit != 3 {
		t.Fatalf("profile identity fields changed: %#v", profile)
	}
	if profile.BaseURL != "https://new.example" || profile.APIMode != string(client.APIModeResponses) || profile.RequestPolicy != string(client.RequestPolicyCompat) {
		t.Fatalf("profile config not updated: %#v", profile)
	}
	if next.Settings.OutputFormat != "jpeg" || next.Settings.OutputDir != "/tmp/images" || next.Settings.ProxyMode != client.ProxyModeNone {
		t.Fatalf("settings not updated: %#v", next.Settings)
	}
	if next.Settings.Theme != "system" || next.Settings.FontScale != 1 {
		t.Fatalf("default visual settings not set: %#v", next.Settings)
	}
}

func TestHistoryItemFromRunUsesWebViewCompatibleFields(t *testing.T) {
	item := HistoryItemFromRun(kernel.Config{
		Prompt:         "cat",
		Mode:           client.ModeEdit,
		Size:           "1024x1536",
		Quality:        "high",
		OutputFormat:   "png",
		Seed:           42,
		NegativePrompt: "blur",
	}, kernel.Result{
		SavedPath:     "/tmp/images/cat.png",
		RawPath:       "/tmp/log/raw.txt",
		RevisedPrompt: "cat revised",
	}, 1.25)
	if item.ID == "" || item.CreatedAt == 0 {
		t.Fatalf("missing identity fields: %#v", item)
	}
	if item.Prompt != "cat" || item.Mode != string(client.ModeEdit) || item.SavedPath != "/tmp/images/cat.png" || item.RawPath != "/tmp/log/raw.txt" {
		t.Fatalf("history item not mapped: %#v", item)
	}
	if !item.PreviewOnly || item.ElapsedSec != 1.25 || item.Seed != 42 || item.NegativePrompt != "blur" {
		t.Fatalf("history metadata not mapped: %#v", item)
	}
}
