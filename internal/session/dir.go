package session

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetSessionsDir returns the directory for storing session databases.
func GetSessionsDir() string {
	if dir := os.Getenv("CRAWLER_SESSIONS_DIR"); dir != "" {
		return dir
	}

	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "crawler", "sessions")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "crawler-sessions")
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, ".config", "crawler", "sessions")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "crawler", "sessions")
	default:
		return filepath.Join(home, ".config", "crawler", "sessions")
	}
}
