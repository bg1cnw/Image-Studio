package client

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildEditsMultipartSetsMaskMimeType(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.png")
	if err := os.WriteFile(src, fakePNG, 0o644); err != nil {
		t.Fatal(err)
	}

	buf, contentType, err := buildEditsMultipart(
		[]string{src},
		base64.StdEncoding.EncodeToString(fakePNG),
		"edit this",
		"gpt-image-2",
		"1024x1024",
		"auto",
		"png",
		"",
		0,
		RequestPolicyOpenAI,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(buf, params["boundary"])
	foundMask := false
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if part.FormName() == "mask" {
			foundMask = true
			if got := part.Header.Get("Content-Type"); got != "image/png" {
				t.Fatalf("mask content-type = %q, want image/png", got)
			}
		}
		_, _ = io.Copy(io.Discard, part)
	}
	if !foundMask {
		t.Fatal("expected mask part in multipart body")
	}
}

func TestBuildEditsMultipartOmitsMaskWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.png")
	if err := os.WriteFile(src, fakePNG, 0o644); err != nil {
		t.Fatal(err)
	}

	buf, _, err := buildEditsMultipart(
		[]string{src},
		"",
		"edit this",
		"gpt-image-2",
		"1024x1024",
		"auto",
		"png",
		"",
		0,
		RequestPolicyOpenAI,
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), `name="mask"`) {
		t.Fatal("multipart body should omit mask part when mask is empty")
	}
}
