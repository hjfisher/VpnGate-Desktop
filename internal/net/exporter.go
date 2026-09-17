package net

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"vpngate/internal/data"
)

// ExportOutcome reports how many configs were written and the target folder.
type ExportOutcome struct {
	Count  int
	Folder string
	Error  error
}

// ExportOVPN writes each server's config as an .ovpn file into folder,
// skipping entries without a decodable config.
func ExportOVPN(folder string, servers []data.VpnServer) ExportOutcome {
	err := os.MkdirAll(folder, 0o755)
	if err != nil {
		return ExportOutcome{Error: err}
	}

	saved := 0
	for _, s := range servers {
		config := s.OpenVPNConfig()
		if config == "" {
			continue
		}
		path := filepath.Join(folder, s.SanitizeFileName())
		if err := os.WriteFile(path, []byte(config), 0o644); err != nil {
			continue
		}
		saved++
	}
	if saved == 0 {
		return ExportOutcome{Count: 0, Folder: folder, Error: fmt.Errorf("no readable configs to export")}
	}
	return ExportOutcome{Count: saved, Folder: folder}
}

// DefaultExportFolder returns the platform Downloads/VPNGate directory.
func DefaultExportFolder() string {
	home, err := os.UserHomeDir()
	if err != nil {
		dir, _ := os.UserConfigDir()
		if dir == "" {
			dir = "."
		}
		return filepath.Join(dir, "VPNGate")
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "Downloads", "VPNGate")
	case "darwin":
		return filepath.Join(home, "Downloads", "VPNGate")
	default:
		// Linux usually has ~/Downloads; fall back gracefully.
		dl := filepath.Join(home, "Downloads")
		if fi, err := os.Stat(dl); err == nil && fi.IsDir() {
			return filepath.Join(dl, "VPNGate")
		}
		return filepath.Join(home, "VPNGate")
	}
}

// Removed isWindows, isMac, fileExists - use runtime.GOOS instead