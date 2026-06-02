package compat

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const SchemaVersion = 1

type State struct {
	SchemaVersion int               `json:"schemaVersion"`
	Client        string            `json:"client,omitempty"`
	UpdatedAt     int64             `json:"updatedAt"`
	Settings      Settings          `json:"settings"`
	Profiles      []UpstreamProfile `json:"profiles"`
	ActiveProfile string            `json:"activeProfileId"`
	History       []HistoryItem     `json:"history"`
	HistoryFull   []HistoryFullItem `json:"historyFull,omitempty"`
}

type Settings struct {
	ProxyMode            string   `json:"proxyMode,omitempty"`
	ProxyURL             string   `json:"proxyURL,omitempty"`
	Theme                string   `json:"theme,omitempty"`
	FontScale            float64  `json:"fontScale,omitempty"`
	OutputFormat         string   `json:"outputFormat,omitempty"`
	OutputDir            string   `json:"outputDir,omitempty"`
	PromptHistory        []string `json:"promptHistory,omitempty"`
	Presets              []Preset `json:"presets,omitempty"`
	KernelRuntimeMode    string   `json:"kernelRuntimeMode,omitempty"`
	TrustedOutputRoots   []string `json:"trustedOutputRoots,omitempty"`
	SavePromptSuppressed bool     `json:"savePromptSuppressed,omitempty"`
}

type UpstreamProfile struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	APIMode          string `json:"apiMode"`
	RequestPolicy    string `json:"requestPolicy"`
	ImagesNewAPICompat bool   `json:"imagesNewAPICompat,omitempty"`
	BaseURL          string `json:"baseURL"`
	TextModelID      string `json:"textModelID"`
	ImageModelID     string `json:"imageModelID"`
	ConcurrencyLimit int    `json:"concurrencyLimit"`
	CreatedAt        int64  `json:"createdAt"`
	LastUsedAt       int64  `json:"lastUsedAt,omitempty"`
}

type Preset struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Size              string `json:"size"`
	Quality           string `json:"quality"`
	OutputFormat      string `json:"outputFormat,omitempty"`
	NegativePrompt    string `json:"negativePrompt"`
	KernelRuntimeMode string `json:"kernelRuntimeMode,omitempty"`
	BatchCount        int    `json:"batchCount"`
}

type HistoryItem struct {
	ID             string  `json:"id"`
	ImageID        string  `json:"imageId,omitempty"`
	PreviewURL     string  `json:"previewUrl,omitempty"`
	FullURL        string  `json:"fullUrl,omitempty"`
	ThumbPath      string  `json:"thumbPath,omitempty"`
	PreviewWidth   int     `json:"previewWidth,omitempty"`
	PreviewHeight  int     `json:"previewHeight,omitempty"`
	ImageB64       string  `json:"imageB64,omitempty"`
	PreviewOnly    bool    `json:"previewOnly,omitempty"`
	Prompt         string  `json:"prompt"`
	RevisedPrompt  string  `json:"revisedPrompt,omitempty"`
	Mode           string  `json:"mode"`
	Size           string  `json:"size"`
	Quality        string  `json:"quality"`
	OutputFormat   string  `json:"outputFormat,omitempty"`
	ParentID       string  `json:"parentId,omitempty"`
	CreatedAt      int64   `json:"createdAt"`
	Seed           int64   `json:"seed,omitempty"`
	NegativePrompt string  `json:"negativePrompt,omitempty"`
	StyleTag       string  `json:"styleTag,omitempty"`
	BatchIndex     int     `json:"batchIndex,omitempty"`
	ElapsedSec     float64 `json:"elapsedSec,omitempty"`
	SavedPath      string  `json:"savedPath,omitempty"`
	RawPath        string  `json:"rawPath,omitempty"`
}

type HistoryFullItem struct {
	ID       string `json:"id"`
	ImageB64 string `json:"imageB64"`
}

func StatePath(stableDataRoot string) string {
	return filepath.Join(stableDataRoot, "compat", "state.json")
}

func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return EmptyState(), nil
		}
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return Normalize(state), nil
}

func Save(path string, state State) error {
	state = Normalize(state)
	if state.UpdatedAt <= 0 {
		state.UpdatedAt = time.Now().UnixMilli()
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func EmptyState() State {
	return State{
		SchemaVersion: SchemaVersion,
		UpdatedAt:     0,
		Profiles:      []UpstreamProfile{},
		History:       []HistoryItem{},
	}
}

func Normalize(state State) State {
	if state.SchemaVersion <= 0 {
		state.SchemaVersion = SchemaVersion
	}
	if state.Profiles == nil {
		state.Profiles = []UpstreamProfile{}
	}
	if state.History == nil {
		state.History = []HistoryItem{}
	}
	return state
}
