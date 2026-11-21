// Package domain holds LearnHarness record types and pure parsing logic.
// It has no Bubble Tea imports so it can be unit-tested in isolation.
package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ResolveDataDir returns the LearnHarness data directory. The explicit
// override wins, then LH_DATA_DIR, then the platform default that the
// desktop app and the learning-data skill both use.
func ResolveDataDir(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	if env := os.Getenv("LH_DATA_DIR"); env != "" {
		return filepath.Abs(env)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	defaultDir := platformDefault(home)
	// The dotfiles deployment keeps the record in a synced checkout. A shell
	// without LH_DATA_DIR (tmux, cron, a GUI launcher) should still find it
	// rather than pointing at an empty OS-default directory.
	synced := filepath.Join(home, "learning", "learnharness-data")
	if !DataDirExists(defaultDir) && DataDirExists(synced) {
		return synced, nil
	}
	return defaultDir, nil
}

func platformDefault(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "LearnHarness", "data")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "LearnHarness", "data")
		}
		return filepath.Join(home, "AppData", "Roaming", "LearnHarness", "data")
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "LearnHarness", "data")
		}
		return filepath.Join(home, ".config", "LearnHarness", "data")
	}
}

// DataDirExists reports whether dir looks like a LearnHarness data root. A
// fresh profile legitimately has neither chats/ nor graph/ yet, so only the
// directory itself is required.
func DataDirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}
