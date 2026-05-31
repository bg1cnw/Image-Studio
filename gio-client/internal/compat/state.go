package compat

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"image-studio/gio-client/internal/kernel"
	shared "image-studio/shared/compat"

	"github.com/yuanhua/image-gptcodex/pkg/client"
	keyring "github.com/zalando/go-keyring"
)

const keyringServiceName = "Image Studio"

func LoadState() (shared.State, string, error) {
	root, err := StableDataRoot()
	if err != nil {
		return shared.EmptyState(), "", err
	}
	path := shared.StatePath(root)
	state, err := shared.Load(path)
	return state, path, err
}

func SaveState(state shared.State) error {
	root, err := StableDataRoot()
	if err != nil {
		return err
	}
	state.Client = "gio"
	if state.UpdatedAt <= 0 {
		state.UpdatedAt = time.Now().UnixMilli()
	}
	return shared.Save(shared.StatePath(root), state)
}

func ConfigFromState(cfg kernel.Config, state shared.State) kernel.Config {
	cfg.OutputDir = DefaultOutputDir()
	if strings.TrimSpace(state.Settings.OutputDir) != "" {
		cfg.OutputDir = state.Settings.OutputDir
	}
	if strings.TrimSpace(state.Settings.OutputFormat) != "" {
		cfg.OutputFormat = state.Settings.OutputFormat
	}
	if strings.TrimSpace(state.Settings.ProxyMode) != "" {
		cfg.ProxyMode = state.Settings.ProxyMode
	}
	cfg.ProxyURL = state.Settings.ProxyURL

	profile, ok := ActiveProfile(state)
	if !ok {
		return cfg
	}
	cfg.BaseURL = profile.BaseURL
	cfg.TextModelID = profile.TextModelID
	cfg.ImageModelID = profile.ImageModelID
	cfg.APIMode = normaliseAPIMode(profile.APIMode)
	cfg.RequestPolicy = normalisePolicy(profile.RequestPolicy)
	cfg.APIKey, _ = ReadAPIKey(profile.ID)
	return cfg
}

func ActiveProfile(state shared.State) (shared.UpstreamProfile, bool) {
	if len(state.Profiles) == 0 {
		return shared.UpstreamProfile{}, false
	}
	if strings.TrimSpace(state.ActiveProfile) != "" {
		for _, profile := range state.Profiles {
			if profile.ID == state.ActiveProfile {
				return profile, true
			}
		}
	}
	return state.Profiles[0], true
}

func SaveConfigAndHistory(cfg kernel.Config, result kernel.Result, elapsedSec float64) error {
	state, _, err := LoadState()
	if err != nil {
		return err
	}
	state = UpsertConfig(state, cfg)
	if strings.TrimSpace(result.SavedPath) != "" {
		item := HistoryItemFromRun(cfg, result, elapsedSec)
		state.History = mergeHistory(item, state.History)
	}
	state.UpdatedAt = time.Now().UnixMilli()
	return SaveState(state)
}

func SaveConfig(cfg kernel.Config) error {
	state, _, err := LoadState()
	if err != nil {
		return err
	}
	state = UpsertConfig(state, cfg)
	state.UpdatedAt = time.Now().UnixMilli()
	return SaveState(state)
}

func SavePromptSuppressed(state shared.State) bool {
	return state.Settings.SavePromptSuppressed
}

func SetSavePromptSuppressed(value bool) error {
	state, _, err := LoadState()
	if err != nil {
		return err
	}
	state.Settings.SavePromptSuppressed = value
	state.UpdatedAt = time.Now().UnixMilli()
	return SaveState(state)
}

func UpsertConfig(state shared.State, cfg kernel.Config) shared.State {
	state = shared.Normalize(state)
	now := time.Now().UnixMilli()
	profileID := strings.TrimSpace(state.ActiveProfile)
	profileIndex := -1
	if profileID != "" {
		for i := range state.Profiles {
			if state.Profiles[i].ID == profileID {
				profileIndex = i
				break
			}
		}
	}
	if profileIndex < 0 && len(state.Profiles) > 0 {
		profileIndex = 0
		profileID = state.Profiles[0].ID
	}
	if profileID == "" {
		profileID = "gio-" + randomID()
	}
	profile := shared.UpstreamProfile{
		ID:               profileID,
		Name:             nextDefaultProfileName(state.Profiles),
		APIMode:          string(normaliseAPIMode(string(cfg.APIMode))),
		RequestPolicy:    string(normalisePolicy(string(cfg.RequestPolicy))),
		BaseURL:          strings.TrimSpace(cfg.BaseURL),
		TextModelID:      strings.TrimSpace(cfg.TextModelID),
		ImageModelID:     strings.TrimSpace(cfg.ImageModelID),
		ConcurrencyLimit: 0,
		CreatedAt:        now,
		LastUsedAt:       now,
	}
	if profileIndex >= 0 {
		profile.Name = state.Profiles[profileIndex].Name
		profile.CreatedAt = state.Profiles[profileIndex].CreatedAt
		profile.ConcurrencyLimit = state.Profiles[profileIndex].ConcurrencyLimit
		state.Profiles[profileIndex] = profile
	} else {
		state.Profiles = append(state.Profiles, profile)
	}
	state.ActiveProfile = profile.ID
	state.Settings.ProxyMode = cfg.ProxyMode
	state.Settings.ProxyURL = strings.TrimSpace(cfg.ProxyURL)
	state.Settings.OutputFormat = strings.TrimSpace(cfg.OutputFormat)
	state.Settings.OutputDir = strings.TrimSpace(cfg.OutputDir)
	if state.Settings.Theme == "" {
		state.Settings.Theme = "system"
	}
	if state.Settings.FontScale == 0 {
		state.Settings.FontScale = 1
	}
	if cfg.APIKey != "" {
		_ = WriteAPIKey(profile.ID, cfg.APIKey)
	}
	return state
}

func HistoryItemFromRun(cfg kernel.Config, result kernel.Result, elapsedSec float64) shared.HistoryItem {
	return shared.HistoryItem{
		ID:             randomID(),
		Prompt:         cfg.Prompt,
		RevisedPrompt:  result.RevisedPrompt,
		Mode:           string(cfg.Mode),
		Size:           cfg.Size,
		Quality:        cfg.Quality,
		OutputFormat:   cfg.OutputFormat,
		CreatedAt:      time.Now().UnixMilli(),
		Seed:           cfg.Seed,
		NegativePrompt: cfg.NegativePrompt,
		ElapsedSec:     elapsedSec,
		SavedPath:      result.SavedPath,
		RawPath:        result.RawPath,
		PreviewOnly:    true,
	}
}

func ReadAPIKey(profileID string) (string, error) {
	value, err := keyring.Get(keyringServiceName, "api-key:profile:"+profileID)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return value, err
}

func WriteAPIKey(profileID, value string) error {
	value = strings.TrimSpace(value)
	user := "api-key:profile:" + profileID
	if value == "" {
		err := keyring.Delete(keyringServiceName, user)
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	return keyring.Set(keyringServiceName, user, value)
}

func mergeHistory(item shared.HistoryItem, items []shared.HistoryItem) []shared.HistoryItem {
	out := make([]shared.HistoryItem, 0, min(len(items)+1, 120))
	seen := map[string]struct{}{item.ID: {}}
	out = append(out, item)
	for _, existing := range items {
		if existing.ID == "" {
			continue
		}
		if _, ok := seen[existing.ID]; ok {
			continue
		}
		seen[existing.ID] = struct{}{}
		out = append(out, existing)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt > out[j].CreatedAt
	})
	if len(out) > 120 {
		out = out[:120]
	}
	return out
}

func normaliseAPIMode(mode string) client.APIMode {
	if mode == string(client.APIModeImages) {
		return client.APIModeImages
	}
	return client.APIModeResponses
}

func normalisePolicy(policy string) client.RequestPolicy {
	if policy == string(client.RequestPolicyCompat) {
		return client.RequestPolicyCompat
	}
	return client.RequestPolicyOpenAI
}

func nextDefaultProfileName(profiles []shared.UpstreamProfile) string {
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

func randomID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}
