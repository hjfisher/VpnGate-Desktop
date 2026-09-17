package net

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"vpngate/internal/data"
)

// OpenFile opens a file (or folder) with the OS default application so the
// .ovpn profile can be imported into any installed OpenVPN client.
func OpenFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	err := cmd.Start()
	if err != nil {
		return err
	}
	// Detach: we intentionally don't Wait() so the app doesn't block on the
	// external process. Child holds no locks we need.
	return nil
}

// SaveConfig writes one server's config into folder and returns its path.
func SaveConfig(folder string, server data.VpnServer) (string, error) {
	config := server.OpenVPNConfig()
	if config == "" {
		return "", os.ErrInvalid
	}
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(folder, server.SanitizeFileName())
	if err := os.WriteFile(path, []byte(config), 0o644); err != nil {
		return "", err
	}
	return path, nil
}