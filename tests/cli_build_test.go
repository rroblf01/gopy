package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCLIBuildSubcommand exercises `gopy build` end-to-end: it builds the
// gopy CLI itself, points it at a fixture, and checks the produced binary
// runs and prints the expected stdout.
func TestCLIBuildSubcommand(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()

	gopyBin := filepath.Join(tmp, "gopy")
	build := exec.Command("go", "build", "-o", gopyBin, "./cmd/gopy")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build gopy: %v\n%s", err, out)
	}

	fixture := filepath.Join(root, "tests", "fixtures", "fib.py")
	wantPy := runPython(t, fixture)

	outBin := filepath.Join(tmp, "fib")
	cmd := exec.Command(gopyBin, "build", "-o", outBin, fixture)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gopy build: %v\n%s", err, out)
	}

	if _, err := os.Stat(outBin); err != nil {
		t.Fatalf("output binary missing: %v", err)
	}

	var got bytes.Buffer
	run := exec.Command(outBin)
	run.Stdout = &got
	if err := run.Run(); err != nil {
		t.Fatalf("run produced binary: %v", err)
	}
	if got.String() != wantPy {
		t.Fatalf("output mismatch\n--- python ---\n%s--- gopy build binary ---\n%s", wantPy, got.String())
	}
}
