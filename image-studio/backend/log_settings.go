package backend

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	compat "image-studio/shared/compat"
)

func (s *Service) SetKeepLogsEnabled(enabled bool) {
	s.mu.Lock()
	s.keepLogs = enabled
	s.mu.Unlock()
}

func (s *Service) Shutdown(_ context.Context) {
	if err := cleanupLogDirsIfDisabled(s.keepLogsEnabled(), s.managedLogDirs()); err != nil {
		println("Warning:", err.Error())
	}
}

func (s *Service) keepLogsEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keepLogs
}

func (s *Service) loadCompatibilitySettings() {
	path, err := compatibilityStatePath()
	if err != nil {
		return
	}
	state, err := compat.Load(path)
	if err != nil {
		return
	}
	s.syncCompatibilitySettings(state)
}

func (s *Service) syncCompatibilitySettings(state compat.State) {
	s.SetKeepLogsEnabled(state.Settings.KeepLogs)
	if outputDir := strings.TrimSpace(state.Settings.OutputDir); outputDir != "" {
		if abs, err := filepath.Abs(outputDir); err == nil {
			s.mu.Lock()
			s.outputDir = abs
			s.mu.Unlock()
			s.addTrustedOutputRoot(abs)
		}
	}
	for _, root := range state.Settings.TrustedOutputRoots {
		s.addTrustedOutputRoot(root)
	}
}

func (s *Service) managedLogDirs() []string {
	return normalizeRoots(s.allowedRoots(managedRawLogFile))
}

func cleanupLogDirsIfDisabled(keepLogs bool, dirs []string) error {
	if keepLogs {
		return nil
	}
	return removeLogDirs(dirs)
}

func removeLogDirs(dirs []string) error {
	var firstErr error
	for _, dir := range normalizeRoots(dirs) {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		if err := os.RemoveAll(dir); err != nil && !errors.Is(err, os.ErrNotExist) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
