package client

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var fakePNG = []byte("\x89PNG\r\n\x1a\nfake")

func mustDecodePayload(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("payload not valid JSON: %v\n%s", err, raw)
	}
	return v
}

func TestBuildPayloadUsesSizeAndQuality(t *testing.T) {
	raw, err := BuildPayload(Options{
		Prompt:  "生成海报",
		Size:    "1536x1024",
		Quality: "high",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := mustDecodePayload(t, raw)

	tool := v["tools"].([]any)[0].(map[string]any)
	if tool["size"] != "1536x1024" {
		t.Errorf("size = %v, want 1536x1024", tool["size"])
	}
	if tool["quality"] != "high" {
		t.Errorf("quality = %v, want high", tool["quality"])
	}
	if tool["model"] != "gpt-image-2" {
		t.Errorf("model = %v, want gpt-image-2", tool["model"])
	}
	if tool["action"] != "generate" {
		t.Errorf("action = %v, want generate", tool["action"])
	}
	if v["stream"] != true {
		t.Errorf("stream = %v, want true", v["stream"])
	}

	input := v["input"].([]any)[0].(map[string]any)
	content := input["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("generate-mode content len = %d, want 1", len(content))
	}
	first := content[0].(map[string]any)
	if first["type"] != "input_text" || first["text"] != "生成海报" {
		t.Errorf("input_text = %v", first)
	}
}

func TestBuildPayloadEditModeAppendsInputImage(t *testing.T) {
	imageURL := "data:image/png;base64,abc123"
	raw, err := BuildPayload(Options{
		Prompt:       "把这张图片改成金色科技风",
		Size:         "1024x1024",
		Quality:      "auto",
		ImageDataURL: imageURL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := mustDecodePayload(t, raw)

	tool := v["tools"].([]any)[0].(map[string]any)
	if tool["action"] != "edit" {
		t.Errorf("action = %v, want edit", tool["action"])
	}
	content := v["input"].([]any)[0].(map[string]any)["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("edit-mode content len = %d, want 2", len(content))
	}
	first := content[0].(map[string]any)
	if first["type"] != "input_text" || first["text"] != "把这张图片改成金色科技风" {
		t.Errorf("input_text = %v", first)
	}
	second := content[1].(map[string]any)
	if second["type"] != "input_image" || second["image_url"] != imageURL {
		t.Errorf("input_image = %v", second)
	}
}

func TestBuildPayloadEmptyPromptError(t *testing.T) {
	_, err := BuildPayload(Options{Prompt: "  "})
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
}

func TestBuildPayloadMultiImageReferences(t *testing.T) {
	urls := []string{
		"data:image/png;base64,AAA",
		"data:image/png;base64,BBB",
		"data:image/jpeg;base64,CCC",
	}
	raw, err := BuildPayload(Options{
		Prompt:        "combine these references",
		ImageDataURLs: urls,
	})
	if err != nil {
		t.Fatal(err)
	}
	v := mustDecodePayload(t, raw)
	tool := v["tools"].([]any)[0].(map[string]any)
	if tool["action"] != "edit" {
		t.Errorf("action = %v, want edit when references provided", tool["action"])
	}
	content := v["input"].([]any)[0].(map[string]any)["content"].([]any)
	if len(content) != 1+len(urls) {
		t.Fatalf("content len = %d, want %d (1 text + %d images)", len(content), 1+len(urls), len(urls))
	}
	for i, url := range urls {
		block := content[1+i].(map[string]any)
		if block["type"] != "input_image" {
			t.Errorf("content[%d].type = %v, want input_image", 1+i, block["type"])
		}
		if block["image_url"] != url {
			t.Errorf("content[%d].image_url = %v, want %s", 1+i, block["image_url"], url)
		}
	}
}

func TestBuildPayloadLegacySingleURLAndMultiCoexist(t *testing.T) {
	raw, err := BuildPayload(Options{
		Prompt:        "mix",
		ImageDataURLs: []string{"data:image/png;base64,AAA"},
		ImageDataURL:  "data:image/png;base64,BBB",
	})
	if err != nil {
		t.Fatal(err)
	}
	v := mustDecodePayload(t, raw)
	content := v["input"].([]any)[0].(map[string]any)["content"].([]any)
	if len(content) != 3 {
		t.Fatalf("expected 1 text + 2 images, got %d blocks", len(content))
	}
}

func TestBuildPayloadOmitsMaskWhenEmpty(t *testing.T) {
	raw, _ := BuildPayload(Options{Prompt: "x"})
	if strings.Contains(string(raw), `"mask"`) {
		t.Errorf("payload should not contain mask field when MaskB64 is empty:\n%s", raw)
	}
}

func TestBuildPayloadIncludesMaskWhenSet(t *testing.T) {
	raw, _ := BuildPayload(Options{Prompt: "x", MaskB64: "AAAA"})
	v := mustDecodePayload(t, raw)
	tool := v["tools"].([]any)[0].(map[string]any)
	if tool["mask"] != "AAAA" {
		t.Errorf("mask = %v, want AAAA", tool["mask"])
	}
}

func TestImageFileToDataURLEncodesPNG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.png")
	if err := os.WriteFile(src, fakePNG, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ImageFileToDataURL(src)
	if err != nil {
		t.Fatal(err)
	}
	want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(fakePNG)
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestImageFileToDataURLUnsupportedExt(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "x.bmp")
	_ = os.WriteFile(src, fakePNG, 0o644)
	_, err := ImageFileToDataURL(src)
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
	if !strings.Contains(err.Error(), "不支持的图片格式") {
		t.Errorf("error message = %q, want 不支持的图片格式", err)
	}
}

func TestImageFileToDataURLMissingFile(t *testing.T) {
	_, err := ImageFileToDataURL(filepath.Join(t.TempDir(), "nope.png"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Hello World", "hello-world"},
		{"  多 个   空 格  ", "多-个-空-格"},
		{"中文 Mix 123", "中文-mix-123"},
		{"", "image"},
		{"!!!", "image"},
	}
	for _, c := range cases {
		if got := Slugify(c.in, ""); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	cases := []struct {
		in, want string
		wantErr  bool
	}{
		{` "E:\foo.png" `, `E:\foo.png`, false},
		{`'/tmp/x.jpg'`, `/tmp/x.jpg`, false},
		{`  `, "", true},
	}
	for _, c := range cases {
		got, err := NormalizePath(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("NormalizePath(%q) wanted error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizePath(%q) err = %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{500, "500 B"},
		{2048, "2.0 KB"},
		{int64(5 * 1024 * 1024), "5.0 MB"},
	}
	for _, c := range cases {
		if got := FormatBytes(c.in); got != c.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
