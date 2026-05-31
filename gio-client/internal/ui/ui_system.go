package ui

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

func chooseImageFiles() ([]string, error) {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command(
			"powershell",
			"-NoProfile",
			"-Command",
			`Add-Type -AssemblyName System.Windows.Forms; `+
				`$dlg = New-Object System.Windows.Forms.OpenFileDialog; `+
				`$dlg.Multiselect = $true; `+
				`$dlg.Filter = "Images|*.png;*.jpg;*.jpeg;*.webp"; `+
				`if ($dlg.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::OutputEncoding = [System.Text.UTF8Encoding]::UTF8; $dlg.FileNames -join "`+"`n"+`" }`,
		)
		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		return parseDialogPaths(string(out)), nil
	default:
		if path, err := exec.LookPath("zenity"); err == nil {
			out, err := exec.Command(path, "--file-selection", "--multiple", "--separator=\n", "--file-filter=Images | *.png *.jpg *.jpeg *.webp").Output()
			if err != nil {
				return nil, err
			}
			return parseDialogPaths(string(out)), nil
		}
		if path, err := exec.LookPath("kdialog"); err == nil {
			out, err := exec.Command(path, "--getopenfilename", ".", "Images (*.png *.jpg *.jpeg *.webp)", "--multiple", "--separate-output").Output()
			if err != nil {
				return nil, err
			}
			return parseDialogPaths(string(out)), nil
		}
		return nil, fmt.Errorf("当前系统没有可用的文件选择器")
	}
}

func parseDialogPaths(raw string) []string {
	lines := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	out := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, `"'`)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return out
}

func openPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("路径为空")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func openExternalURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("URL 无效: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https URL")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", parsed.String()).Start()
	case "darwin":
		return exec.Command("open", parsed.String()).Start()
	default:
		return exec.Command("xdg-open", parsed.String()).Start()
	}
}
