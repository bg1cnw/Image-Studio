package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// startSSEServer returns an httptest.Server that streams the given lines as SSE.
// Each line is sent with a `data: ` prefix and flushed; CR/LF added between.
func startSSEServer(t *testing.T, eventLines []string, statusCode int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(statusCode)
		flusher, _ := w.(http.Flusher)
		for _, ln := range eventLines {
			fmt.Fprintln(w, ln)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// pointTransportAtServer creates a NativeTransport that rewrites the URL
// passed in Request to srv.URL while preserving headers/body. For tests we
// just inject the server URL directly into Request.URL via the caller.
func TestRequestAndExtractWithRetries_HappyPath(t *testing.T) {
	pngB64 := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nfake"))
	ev := func(m map[string]any) string {
		b, _ := json.Marshal(m)
		return "data: " + string(b)
	}
	lines := []string{
		ev(map[string]any{"type": "response.created"}),
		ev(map[string]any{
			"type": "response.output_item.done",
			"item": map[string]any{
				"type":   "image_generation_call",
				"result": pngB64,
			},
		}),
	}
	srv := startSSEServer(t, lines, http.StatusOK)

	transport := &injectingTransport{
		inner: &NativeTransport{},
		url:   srv.URL,
	}

	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, rawPath, err := RequestAndExtractWithRetries(
		ctx, transport,
		Options{APIKey: "sk-test", Prompt: "hello", Size: "1024x1024", Quality: "auto"},
		dir, "20260518-200000",
		nil, nil,
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if res.ImageB64 != pngB64 {
		t.Errorf("image b64 mismatch")
	}
	if !strings.HasSuffix(rawPath, "-attempt1.txt") {
		t.Errorf("rawPath = %q, expected attempt1", rawPath)
	}
	// Raw response file should exist and contain the image's base64.
	body, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), pngB64) {
		t.Errorf("raw response file missing image base64")
	}
}

func TestRequestAndExtractWithRetries_RetryOn524(t *testing.T) {
	// First attempt returns Cloudflare 524 HTML (retryable); second attempt succeeds.
	pngB64 := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nfake"))
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if hits == 1 {
			fmt.Fprint(w, "<html>Error code 524 | 524: A timeout occurred</html>")
			return
		}
		fmt.Fprintln(w, `data: {"type":"response.created"}`)
		body, _ := json.Marshal(map[string]any{
			"type": "response.output_item.done",
			"item": map[string]any{
				"type":   "image_generation_call",
				"result": pngB64,
			},
		})
		fmt.Fprintln(w, "data: "+string(body))
	}))
	defer srv.Close()

	// Speed up retry backoff for the test by overriding via env-like indirection.
	// (We rely on the fact that the implementation reads time.Sleep against a
	// constant; rather than complicate prod code, we just accept a 15s wait.)
	// To keep test under timeout, override the constant via a build tag would
	// be cleaner. For now we shrink with a global hack: scope the test under
	// a Go flag t.Setenv won't see, so just wrap with longer timeout.

	// Shrink backoff for fast test execution.
	original := RetryBackoffSeconds
	RetryBackoffSeconds = 0
	t.Cleanup(func() { RetryBackoffSeconds = original })

	transport := &injectingTransport{inner: &NativeTransport{}, url: srv.URL}
	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, _, err := RequestAndExtractWithRetries(
		ctx, transport,
		Options{APIKey: "sk-test", Prompt: "p", Size: "1024x1024", Quality: "auto"},
		dir, "20260518-200001",
		nil, nil,
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if res.ImageB64 != pngB64 {
		t.Errorf("image b64 mismatch on retry path")
	}
	if hits != 2 {
		t.Errorf("hits = %d, want 2", hits)
	}
}

func TestRequestAndExtractContextCancel(t *testing.T) {
	// Server hangs forever; ensure ctx cancellation propagates.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		// Stream one event then block.
		fmt.Fprintln(w, `data: {"type":"response.created"}`)
		if flusher != nil {
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	transport := &injectingTransport{inner: &NativeTransport{}, url: srv.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	dir := t.TempDir()
	rawPath := filepath.Join(dir, "raw.txt")
	f, _ := os.Create(rawPath)
	_, err := RequestAndExtract(ctx, transport, Options{APIKey: "k", Prompt: "p"}, f, nil)
	f.Close()
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

// injectingTransport rewrites the URL on the request before delegating.
type injectingTransport struct {
	inner Transport
	url   string
}

func (i *injectingTransport) Stream(ctx context.Context, req Request, rawSink io.Writer, progress chan<- string) error {
	req.URL = i.url + "/v1/responses"
	return i.inner.Stream(ctx, req, rawSink, progress)
}
