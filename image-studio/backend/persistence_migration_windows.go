//go:build windows

package backend

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const windowsWebviewMigrationMarker = ".migrated-to-image-studio-documents"

func MigrateWindowsWebviewDataDir(dst, legacy string) error {
	return MigrateWindowsWebviewDataDirs(dst, []string{legacy})
}

func MigrateWindowsWebviewDataDirs(dst string, legacyPaths []string) error {
	dst = strings.TrimSpace(dst)
	if dst == "" || len(legacyPaths) == 0 {
		return nil
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	if dirExists(dstAbs) {
		return nil
	}
	candidates := windowsWebviewMigrationCandidates(dstAbs, legacyPaths)
	if len(candidates) == 0 {
		return nil
	}
	legacyAbs := candidates[0].path
	if err := os.MkdirAll(filepath.Dir(dstAbs), secureDirMode); err != nil {
		return err
	}
	if err := os.Rename(legacyAbs, dstAbs); err == nil {
		return nil
	}
	if err := copyDir(legacyAbs, dstAbs); err != nil {
		return err
	}
	marker := filepath.Join(legacyAbs, windowsWebviewMigrationMarker)
	return os.WriteFile(marker, []byte(dstAbs), secureFileMode)
}

type windowsWebviewMigrationCandidate struct {
	path  string
	score int64
}

func windowsWebviewMigrationCandidates(dstAbs string, legacyPaths []string) []windowsWebviewMigrationCandidate {
	candidates := make([]windowsWebviewMigrationCandidate, 0, len(legacyPaths))
	seen := map[string]struct{}{}
	for _, legacy := range legacyPaths {
		legacy = strings.TrimSpace(legacy)
		if legacy == "" {
			continue
		}
		legacyAbs, err := filepath.Abs(legacy)
		if err != nil || samePath(dstAbs, legacyAbs) {
			continue
		}
		key := strings.ToLower(filepath.Clean(legacyAbs))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if !dirExists(legacyAbs) {
			continue
		}
		score := webviewProfileScore(legacyAbs)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, windowsWebviewMigrationCandidate{path: legacyAbs, score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	return candidates
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func webviewProfileScore(path string) int64 {
	var score int64
	for _, rel := range []string{"IndexedDB", "Local Storage"} {
		score += dirSize(filepath.Join(path, rel))
	}
	for _, rel := range []string{"Preferences", filepath.Join("Default", "Preferences")} {
		if info, err := os.Stat(filepath.Join(path, rel)); err == nil && !info.IsDir() {
			score += info.Size()
		}
	}
	if score > 0 {
		return score
	}
	for _, rel := range []string{"Default", "Network"} {
		if _, err := os.Stat(filepath.Join(path, rel)); err == nil {
			return 1
		}
	}
	return 0
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.WalkDir(path, func(walkPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err == nil {
			size += info.Size()
		}
		_ = walkPath
		return nil
	})
	return size
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, secureDirMode)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info)
	})
}

func copyFile(src, dst string, info fs.FileInfo) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), secureDirMode); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	mtime := info.ModTime()
	if mtime.IsZero() {
		mtime = time.Now()
	}
	return os.Chtimes(dst, mtime, mtime)
}
