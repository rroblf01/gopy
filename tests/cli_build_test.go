package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// buildGopyCLI compiles cmd/gopy into a temp dir and returns the binary
// path. Shared across CLI tests.
func buildGopyCLI(t *testing.T, root string) string {
	t.Helper()
	tmp := t.TempDir()
	gopyBin := filepath.Join(tmp, "gopy")
	build := exec.Command("go", "build", "-o", gopyBin, "./cmd/gopy")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build gopy: %v\n%s", err, out)
	}
	return gopyBin
}

// TestCLIBuildSubcommand exercises `gopy build` end-to-end: it builds the
// gopy CLI itself, points it at a fixture, and checks the produced binary
// runs and prints the expected stdout.
func TestCLIBuildSubcommand(t *testing.T) {
	root := repoRoot(t)
	gopyBin := buildGopyCLI(t, root)
	tmp := t.TempDir()

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

// TestCLIBuildSharedRuntime verifies that a multi-file project where two
// sources both trigger the same runtime helper (here `print`) builds
// cleanly. Without the shared gopy_runtime.go consolidation, each file
// would emit its own copy of `__gopy_print` and Go would reject the
// package with "__gopy_print redeclared in this block".
func TestCLIBuildSharedRuntime(t *testing.T) {
	root := repoRoot(t)
	gopyBin := buildGopyCLI(t, root)
	tmp := t.TempDir()
	proj := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"a.py": "def hi(name: str) -> None:\n    print(\"hi\", name)\n",
		"main.py": "from a import hi\n" +
			"def main() -> None:\n    hi(\"ana\")\n    print(\"done\")\n" +
			"if __name__ == \"__main__\":\n    main()\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(proj, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outBin := filepath.Join(tmp, "shared-rt")
	cmd := exec.Command(gopyBin, "build", "-o", outBin, proj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gopy build (shared runtime): %v\n%s", err, out)
	}
	var got bytes.Buffer
	run := exec.Command(outBin)
	run.Stdout = &got
	if err := run.Run(); err != nil {
		t.Fatalf("run binary: %v", err)
	}
	want := "hi ana\ndone\n"
	if got.String() != want {
		t.Fatalf("output mismatch\nwant %q\n got %q", want, got.String())
	}
}

// TestCLIBuildProject runs `gopy build <dir>` against a multi-file fixture
// and checks the resulting binary's stdout matches running `python3 main.py`
// from inside that directory. Covers the directory entry point that lets
// gopy serve as a one-shot transpiler+linker for project layouts.
func TestCLIBuildProject(t *testing.T) {
	root := repoRoot(t)
	gopyBin := buildGopyCLI(t, root)
	tmp := t.TempDir()

	projDir := filepath.Join(root, "tests", "fixtures_multi", "calc")
	wantPy := runPythonInDir(t, projDir, "main.py")

	outBin := filepath.Join(tmp, "calc-bin")
	cmd := exec.Command(gopyBin, "build", "-o", outBin, projDir)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gopy build (project): %v\n%s", err, out)
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
