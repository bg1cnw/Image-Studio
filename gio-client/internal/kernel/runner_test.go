package kernel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/yuanhua/image-gptcodex/pkg/client"
)

func TestParseSourcePaths(t *testing.T) {
	got := ParseSourcePaths(" /tmp/a.png\n'/tmp/b.jpg',\"/tmp/a.png\" ")
	want := []string{"/tmp/a.png", "/tmp/b.jpg"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeConfigDefaults(t *testing.T) {
	cfg := normalizeConfig(Config{
		Prompt:    "  hello  ",
		Mode:      client.Mode("unknown"),
		OutputDir: filepath.Join("tmp", "out"),
	})
	if cfg.Prompt != "hello" {
		t.Fatalf("prompt=%q", cfg.Prompt)
	}
	if cfg.Mode != client.ModeGenerate {
		t.Fatalf("mode=%q", cfg.Mode)
	}
	if cfg.APIMode != client.APIModeResponses {
		t.Fatalf("api mode=%q", cfg.APIMode)
	}
	if cfg.TextModelID == "" || cfg.ImageModelID == "" || cfg.OutputFormat == "" {
		t.Fatalf("missing defaults: %#v", cfg)
	}
}

func TestBuildImageNameMapsJPEGExtension(t *testing.T) {
	got := buildImageName(client.ModeEdit, "A cat wearing sunglasses", "20260531-120000", "jpeg")
	want := "image-edit-a-cat-wearing-sunglasses-20260531-120000.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestProbeUpstreamReturnsModelCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer sk-test" {
			t.Fatalf("authorization=%q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.5"},{"id":"gpt-image-2"}]}`))
	}))
	defer server.Close()

	result, err := ProbeUpstream(context.Background(), Config{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("ProbeUpstream: %v", err)
	}
	if result.ModelCount != 2 {
		t.Fatalf("ModelCount=%d want 2", result.ModelCount)
	}
}
