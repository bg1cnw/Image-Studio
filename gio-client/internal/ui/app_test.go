package ui

import (
	"os"
	"path/filepath"
	"testing"
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
