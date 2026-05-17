package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLocatePythonVenv verifies that LocatePython prefers a sibling
// .venv/bin/python3 over the system interpreter when one is present.
func TestLocatePythonVenv(t *testing.T) {
	dir := t.TempDir()
	venvBin := filepath.Join(dir, ".venv", "bin")
	if err := os.MkdirAll(venvBin, 0o755); err != nil {
		t.Fatal(err)
	}
	pyPath := filepath.Join(venvBin, "python3")
	if err := os.WriteFile(pyPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := LocatePython(filepath.Join(dir, "src.py"))
	want, _ := filepath.Abs(pyPath)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLocatePythonFallback(t *testing.T) {
	// Empty hint and no GOPY_PYTHON → fallback to "python3" on PATH.
	t.Setenv("GOPY_PYTHON", "")
	got := LocatePython("")
	if got != "python3" {
		t.Fatalf("got %q, want python3", got)
	}
}

func TestLocatePythonEnvOverride(t *testing.T) {
	t.Setenv("GOPY_PYTHON", "/usr/local/bin/special-python")
	got := LocatePython("/somewhere/whatever")
	if got != "/usr/local/bin/special-python" {
		t.Fatalf("got %q, want override", got)
	}
}
