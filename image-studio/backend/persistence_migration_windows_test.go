//go:build windows

package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateWindowsWebviewDataDirMovesLegacyProfile(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "image-studio.exe")
	dst := filepath.Join(root, "Image Studio", "webview")
	dbFile := filepath.Join(legacy, "IndexedDB", "image-studio.indexeddb.leveldb", "000003.log")
	if err := os.MkdirAll(filepath.Dir(dbFile), secureDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbFile, []byte("history-db"), secureFileMode); err != nil {
		t.Fatal(err)
	}

	if err := MigrateWindowsWebviewDataDir(dst, legacy); err != nil {
		t.Fatal(err)
	}

	migrated := filepath.Join(dst, "IndexedDB", "image-studio.indexeddb.leveldb", "000003.log")
	data, err := os.ReadFile(migrated)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "history-db" {
		t.Fatalf("migrated data = %q", data)
	}
}

func TestMigrateWindowsWebviewDataDirKeepsExistingDestination(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "image-studio.exe")
	dst := filepath.Join(root, "Image Studio", "webview")
	if err := os.MkdirAll(legacy, secureDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, secureDirMode); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(dst, "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep"), secureFileMode); err != nil {
		t.Fatal(err)
	}

	if err := MigrateWindowsWebviewDataDir(dst, legacy); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("destination was overwritten: %q", data)
	}
}

func TestMigrateWindowsWebviewDataDirSkipsNonProfileDirectory(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "renamed.exe")
	dst := filepath.Join(root, "Image Studio", "webview")
	if err := os.MkdirAll(legacy, secureDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "notes.txt"), []byte("not-webview"), secureFileMode); err != nil {
		t.Fatal(err)
	}

	if err := MigrateWindowsWebviewDataDir(dst, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("expected destination to stay absent, stat err = %v", err)
	}
}

func TestMigrateWindowsWebviewDataDirsPrefersProfileWithStoredData(t *testing.T) {
	root := t.TempDir()
	emptyProfile := filepath.Join(root, "renamed.exe")
	populatedProfile := filepath.Join(root, "image-studio.exe")
	dst := filepath.Join(root, "Image Studio", "webview")

	if err := os.MkdirAll(filepath.Join(emptyProfile, "Network"), secureDirMode); err != nil {
		t.Fatal(err)
	}
	dbFile := filepath.Join(populatedProfile, "IndexedDB", "image-studio.indexeddb.leveldb", "000003.log")
	if err := os.MkdirAll(filepath.Dir(dbFile), secureDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbFile, []byte("real-history"), secureFileMode); err != nil {
		t.Fatal(err)
	}

	if err := MigrateWindowsWebviewDataDirs(dst, []string{emptyProfile, populatedProfile}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "IndexedDB", "image-studio.indexeddb.leveldb", "000003.log"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "real-history" {
		t.Fatalf("migrated data = %q", data)
	}
}
