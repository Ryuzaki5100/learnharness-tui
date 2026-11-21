package domain

import (
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
