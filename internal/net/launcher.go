package net

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"vpngate/internal/data"
)

// OpenFile opens a file (or folder) with the OS default application so the
// .ovpn profile can be imported into any installed OpenVPN client.
// Returns nil if the process started successfully. On Linux, attempts to
// verify the file was actually handled by waiting briefly for xdg-open to
// complete (with a timeout). On other platforms, starts the process detached.
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

	// On Linux, wait briefly for xdg-open to finish to detect if it actually
	// handled the file. Use a short timeout since xdg-open may fork or fail
	// silently if no handler is registered.
	if runtime.GOOS == "linux" {
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case err := <-done:
			if err != nil {
				// xdg-open returned an error - likely no handler registered
				return &LaunchError{
					Path:   path,
					Err:    err,
					IsNoHandler: isNoHandlerError(err),
				}
			}
			return nil
		case <-time.After(2 * time.Second):
			// xdg-open is taking too long; it may have forked or be waiting.
			// Assume it's working and detach.
			return nil
		}
	}

	// On Windows/macOS, detach immediately (process is typically very fast)
	return nil
}

// LaunchError provides details about why a file failed to open.
type LaunchError struct {
	Path       string
	Err        error
	IsNoHandler bool
}

func (e *LaunchError) Error() string {
	if e.IsNoHandler {
		return "no application registered to open .ovpn files"
	}
	return "failed to open file: " + e.Err.Error()
}

func isNoHandlerError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Common error messages when no handler is registered
	return containsAny(errStr, []string{
		"no application",
		"no handler",
		"unable to find",
		"not supported",
		"no default",
	})
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
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