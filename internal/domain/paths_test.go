package domain

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveDataDirPrecedence(t *testing.T) {
	t.Setenv("LH_DATA_DIR", "/tmp/from-env")

	got, err := ResolveDataDir("/tmp/from-flag")
	if err != nil {
		t.Fatalf("resolve override: %v", err)
	}
	if filepath.Clean(got) != "/tmp/from-flag" {
		t.Fatalf("override ignored: %q", got)
	}

	got, err = ResolveDataDir("")
	if err != nil {
		t.Fatalf("resolve env: %v", err)
	}
	if filepath.Clean(got) != "/tmp/from-env" {
		t.Fatalf("env ignored: %q", got)
	}
}

func TestResolveDataDirDefault(t *testing.T) {
	t.Setenv("LH_DATA_DIR", "")
	t.Setenv("HOME", t.TempDir())
	got, err := ResolveDataDir("")
	if err != nil {
		t.Fatalf("resolve default: %v", err)
	}
	if !strings.HasSuffix(got, filepath.Join("LearnHarness", "data")) {
		t.Fatalf("unexpected default %q", got)
	}
	if runtime.GOOS == "darwin" && !strings.Contains(got, filepath.Join("Library", "Application Support")) {
		t.Fatalf("macOS default must use Application Support: %q", got)
	}
}

func TestResolveDataDirFallsBackToSyncedCheckout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LH_DATA_DIR", "")
	synced := filepath.Join(home, "learning", "learnharness-data")
	if err := os.MkdirAll(synced, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got, err := ResolveDataDir("")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != synced {
		t.Fatalf("got %q, want the synced checkout %q", got, synced)
	}
}
