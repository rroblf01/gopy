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

// TestCLIBuildRecursive runs `gopy build <entry.py>` on a single entry file
// that imports a sibling module both qualified (`mod.fn(...)`) and via
// `from mod import NAME`. The build must follow the import to the sibling .py,
// transpile it into the shared package, and strip the qualifier — proving the
// single-file entry point recursively pulls in local pure-Python deps.
func TestCLIBuildRecursive(t *testing.T) {
	root := repoRoot(t)
	gopyBin := buildGopyCLI(t, root)
	tmp := t.TempDir()
	files := map[string]string{
		"helper.py": "GREETING = \"hi\"\n\n" +
			"def shout(s: str) -> str:\n    return s.upper()\n",
		"app.py": "import helper\n" +
			"from helper import GREETING\n" +
			"def main() -> None:\n" +
			"    print(GREETING, helper.shout(\"world\"))\n" +
			"if __name__ == \"__main__\":\n    main()\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outBin := filepath.Join(tmp, "recur-bin")
	cmd := exec.Command(gopyBin, "build", "-o", outBin, filepath.Join(tmp, "app.py"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gopy build (recursive): %v\n%s", err, out)
	}
	var got bytes.Buffer
	run := exec.Command(outBin)
	run.Stdout = &got
	if err := run.Run(); err != nil {
		t.Fatalf("run binary: %v", err)
	}
	want := "hi WORLD\n"
	if got.String() != want {
		t.Fatalf("output mismatch\nwant %q\n got %q", want, got.String())
	}
}

// TestCLIBuildSubpackage runs `gopy build <entry.py>` on an entry that imports
// a sub*package* (`from pkg.mod import f`, `from pkg import mod; mod.g()`), not
// just a flat sibling. The build must resolve `pkg/mod.py` and `pkg/__init__.py`
// under the entry's tree, transpile them into the shared package, and strip the
// `mod.` qualifier — proving recursion follows package layouts, all pure Go.
func TestCLIBuildSubpackage(t *testing.T) {
	root := repoRoot(t)
	gopyBin := buildGopyCLI(t, root)
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "mypkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"mypkg/__init__.py": "",
		"mypkg/calc.py": "def add(a: int, b: int) -> int:\n    return a + b\n\n" +
			"def mul(a: int, b: int) -> int:\n    return a * b\n",
		"main.py": "from mypkg.calc import add\n" +
			"from mypkg import calc\n" +
			"def main() -> None:\n    print(add(2, 3))\n    print(calc.mul(4, 5))\n" +
			"if __name__ == \"__main__\":\n    main()\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outBin := filepath.Join(tmp, "pkg-bin")
	cmd := exec.Command(gopyBin, "build", "-o", outBin, filepath.Join(tmp, "main.py"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gopy build (subpackage): %v\n%s", err, out)
	}
	var got bytes.Buffer
	run := exec.Command(outBin)
	run.Stdout = &got
	if err := run.Run(); err != nil {
		t.Fatalf("run binary: %v", err)
	}
	want := "5\n20\n"
	if got.String() != want {
		t.Fatalf("output mismatch\nwant %q\n got %q", want, got.String())
	}
}

// TestCLIBuildRelativeImports builds an entry that imports a package whose
// modules use relative imports (`from .util import f`, `from . import util`).
// Discovery must resolve those relative to the importing module's own package
// dir, all into pure Go.
func TestCLIBuildRelativeImports(t *testing.T) {
	root := repoRoot(t)
	gopyBin := buildGopyCLI(t, root)
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "mypkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"mypkg/__init__.py": "",
		"mypkg/util.py":     "def helper(s: str) -> str:\n    return s.upper()\n",
		"mypkg/core.py": "from .util import helper\n" +
			"from . import util\n" +
			"def run(s: str) -> str:\n    return helper(s) + \"/\" + util.helper(s)\n",
		"main.py": "from mypkg.core import run\n" +
			"def main() -> None:\n    print(run(\"hi\"))\n" +
			"if __name__ == \"__main__\":\n    main()\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outBin := filepath.Join(tmp, "rel-bin")
	cmd := exec.Command(gopyBin, "build", "-o", outBin, filepath.Join(tmp, "main.py"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gopy build (relative imports): %v\n%s", err, out)
	}
	var got bytes.Buffer
	run := exec.Command(outBin)
	run.Stdout = &got
	if err := run.Run(); err != nil {
		t.Fatalf("run binary: %v", err)
	}
	if want := "HI/HI\n"; got.String() != want {
		t.Fatalf("output mismatch\nwant %q\n got %q", want, got.String())
	}
}

// TestCLIBuildDottedImport builds an entry that uses `import pkg.mod` and then
// the deep-qualified `pkg.mod.fn(...)` / `pkg.mod.CONST` access, which must
// have its module qualifier stripped to a bare package-level symbol.
func TestCLIBuildDottedImport(t *testing.T) {
	root := repoRoot(t)
	gopyBin := buildGopyCLI(t, root)
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "mypkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"mypkg/__init__.py": "",
		"mypkg/calc.py":     "SCALE = 10\n\ndef add(a: int, b: int) -> int:\n    return a + b\n",
		"main.py": "import mypkg.calc\n" +
			"def main() -> None:\n    print(mypkg.calc.add(2, 3))\n    print(mypkg.calc.SCALE)\n" +
			"if __name__ == \"__main__\":\n    main()\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outBin := filepath.Join(tmp, "dot-bin")
	cmd := exec.Command(gopyBin, "build", "-o", outBin, filepath.Join(tmp, "main.py"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gopy build (dotted import): %v\n%s", err, out)
	}
	var got bytes.Buffer
	run := exec.Command(outBin)
	run.Stdout = &got
	if err := run.Run(); err != nil {
		t.Fatalf("run binary: %v", err)
	}
	if want := "5\n10\n"; got.String() != want {
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
