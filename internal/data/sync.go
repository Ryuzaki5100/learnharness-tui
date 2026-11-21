package data

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SyncStatus is the one-line git state shown in the status bar.
type SyncStatus struct {
	Available bool
	State     string
	Detail    string
}

// String renders the compact status-bar form.
func (s SyncStatus) String() string {
	if !s.Available {
		return "sync: unavailable"
	}
	if s.Detail != "" {
		return "sync: " + s.State + " (" + s.Detail + ")"
	}
	if s.State == "" {
		return "sync: unknown"
	}
	return "sync: " + s.State
}

// SyncAvailable reports whether the dotfiles lh-sync helper is installed.
func SyncAvailable() bool {
	_, err := exec.LookPath("lh-sync")
	return err == nil
}

// ReadSyncStatus asks lh-sync for the repository state. It never mutates the
// repository.
func ReadSyncStatus(dataDir string) SyncStatus {
	path, err := exec.LookPath("lh-sync")
	if err != nil {
		return SyncStatus{Available: false}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, "status")
	command.Env = append(command.Environ(), "LH_DATA_DIR="+dataDir)
	out, err := command.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil && text == "" {
		return SyncStatus{Available: true, State: "error", Detail: err.Error()}
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return SyncStatus{Available: true, State: "unknown"}
	}
	status := SyncStatus{Available: true, State: fields[0]}
	if len(fields) > 1 {
		status.Detail = strings.Join(fields[1:], " ")
	}
	return status
}

// RunSync executes `lh-sync <action>` (pull, push, cycle) for the data
// directory and returns its combined output. A refusal is a non-nil error
// with the reported reason, never a silent success.
func RunSync(dataDir, action string) (string, error) {
	path, err := exec.LookPath("lh-sync")
	if err != nil {
		return "", fmt.Errorf("lh-sync is not on PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, path, action)
	command.Env = append(command.Environ(), "LH_DATA_DIR="+dataDir)
	out, err := command.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, fmt.Errorf("lh-sync %s: %s", action, text)
	}
	return text, nil
}
