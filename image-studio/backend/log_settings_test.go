package backend

import (
	"os"
	"path/filepath"
	"testing"

	compat "image-studio/shared/compat"
)

func TestSyncCompatibilitySettingsTracksKeepLogsAndRoots(t *testing.T) {
	svc := NewService()
	outputRoot := filepath.Join(t.TempDir(), "output")
	trustedRoot := filepath.Join(t.TempDir(), "trusted")

	svc.syncCompatibilitySettings(compat.State{
		Settings: compat.Settings{
			OutputDir:          outputRoot,
			TrustedOutputRoots: []string{trustedRoot},
			KeepLogs:           true,
		},
	})

	if !svc.keepLogsEnabled() {
		t.Fatal("expected keepLogs to be enabled")
	}
	if got := svc.currentOutputRootSnapshot(); got != outputRoot {
		t.Fatalf("currentOutputRootSnapshot() = %q, want %q", got, outputRoot)
	}
	roots := svc.trustedOutputRootsSnapshot()
	if len(roots) != 2 {
		t.Fatalf("trustedOutputRootsSnapshot() len = %d, want 2 (%v)", len(roots), roots)
	}
}

func TestCleanupLogDirsIfDisabledRemovesLogDirectories(t *testing.T) {
	logA := filepath.Join(t.TempDir(), "a", "log")
	logB := filepath.Join(t.TempDir(), "b", "log")
	if err := os.MkdirAll(logA, secureDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logB, secureDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logA, "raw.txt"), []byte("raw"), secureFileMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logB, "raw.json"), []byte("{}"), secureFileMode); err != nil {
		t.Fatal(err)
	}

	if err := cleanupLogDirsIfDisabled(false, []string{logA, logB}); err != nil {
		t.Fatalf("cleanupLogDirsIfDisabled() error = %v", err)
	}
	if _, err := os.Stat(logA); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, got %v", logA, err)
	}
	if _, err := os.Stat(logB); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, got %v", logB, err)
	}
}

func TestCleanupLogDirsIfDisabledLeavesLogsWhenEnabled(t *testing.T) {
	logDir := filepath.Join(t.TempDir(), "log")
	if err := os.MkdirAll(logDir, secureDirMode); err != nil {
		t.Fatal(err)
	}
	logFile := filepath.Join(logDir, "raw.txt")
	if err := os.WriteFile(logFile, []byte("raw"), secureFileMode); err != nil {
		t.Fatal(err)
	}

	if err := cleanupLogDirsIfDisabled(true, []string{logDir}); err != nil {
		t.Fatalf("cleanupLogDirsIfDisabled() error = %v", err)
	}
	if _, err := os.Stat(logFile); err != nil {
		t.Fatalf("expected %s to remain, got %v", logFile, err)
	}
}
