package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rroblf01/gopy/ir"
	"github.com/rroblf01/gopy/parser"
	"github.com/rroblf01/gopy/transpile"
)

// repoRoot returns the directory of this test file's parent (the repo root).
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

// transpileFile reads a .py fixture and returns the generated Go source.
func transpileFile(t *testing.T, root, pyPath string) []byte {
	t.Helper()
	dumper := filepath.Join(root, "scripts", "py_ast_dump.py")
	node, err := parser.ParseFile(dumper, pyPath)
	if err != nil {
		t.Fatalf("parse %s: %v", pyPath, err)
	}
	mod, err := ir.Lower(filepath.Base(pyPath), node)
	if err != nil {
		t.Fatalf("lower %s: %v", pyPath, err)
	}
	src, err := transpile.Module(mod, transpile.Options{PackageName: "main"})
	if err != nil {
		t.Fatalf("transpile %s: %v\n%s", pyPath, err, src)
	}
	return src
}

// runPython runs the original .py and returns its stdout.
func runPython(t *testing.T, path string) string {
	t.Helper()
	var out, errBuf bytes.Buffer
	cmd := exec.Command("python3", path)
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("python3 %s: %v: %s", path, err, errBuf.String())
	}
	return out.String()
}

// buildAndRunGo writes Go source to a temp dir, builds it, runs the binary,
// and returns stdout. Errors include the generated source for easy diagnosis.
func buildAndRunGo(t *testing.T, src []byte) string {
	t.Helper()
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goFile, src, 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "prog")
	var errBuf bytes.Buffer
	build := exec.Command("go", "build", "-o", bin, goFile)
	build.Stderr = &errBuf
	if err := build.Run(); err != nil {
		t.Fatalf("go build failed: %v\nstderr: %s\n--- source ---\n%s", err, errBuf.String(), src)
	}
	var out bytes.Buffer
	run := exec.Command(bin)
	run.Stdout = &out
	run.Stderr = &errBuf
	if err := run.Run(); err != nil {
		t.Fatalf("run binary: %v: %s", err, errBuf.String())
	}
	return out.String()
}

// TestMultiFile transpiles every .py in tests/fixtures_multi/<project>/ into
// a shared Go package, builds the resulting program, and compares its stdout
// to running the Python entry point via `python3 main.py` from inside the
// fixture directory (so Python's own import resolution finds sibling modules).
func TestMultiFile(t *testing.T) {
	root := repoRoot(t)
	multiDir := filepath.Join(root, "tests", "fixtures_multi")
	projects, err := os.ReadDir(multiDir)
	if err != nil {
		t.Fatal(err)
	}
	dumper := filepath.Join(root, "scripts", "py_ast_dump.py")
	for _, p := range projects {
		if !p.IsDir() {
			continue
		}
		t.Run(p.Name(), func(t *testing.T) {
			projDir := filepath.Join(multiDir, p.Name())
			wantPy := runPythonInDir(t, projDir, "main.py")

			outDir := t.TempDir()
			// Drop a minimal go.mod so `go build .` works in an isolated tempdir.
			if err := os.WriteFile(filepath.Join(outDir, "go.mod"),
				[]byte("module gopytest\n\ngo 1.22\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			// Transpile every .py in projDir → .go in outDir, package main.
			entries, err := os.ReadDir(projDir)
			if err != nil {
				t.Fatal(err)
			}
			// Every .py stem is a sibling module in the shared package, so
			// qualified `mod.fn(...)` accesses drop the qualifier.
			var localMods []string
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".py") {
					localMods = append(localMods, strings.TrimSuffix(e.Name(), ".py"))
				}
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
					continue
				}
				pyPath := filepath.Join(projDir, e.Name())
				node, err := parser.ParseFile(dumper, pyPath)
				if err != nil {
					t.Fatalf("parse %s: %v", pyPath, err)
				}
				mod, err := ir.Lower(e.Name(), node)
				if err != nil {
					t.Fatalf("lower %s: %v", pyPath, err)
				}
				src, err := transpile.Module(mod, transpile.Options{PackageName: "main", LocalModules: localMods})
				if err != nil {
					t.Fatalf("transpile %s: %v\n%s", pyPath, err, src)
				}
				goFile := filepath.Join(outDir, strings.TrimSuffix(e.Name(), ".py")+".go")
				if err := os.WriteFile(goFile, src, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			bin := filepath.Join(outDir, "prog")
			var errBuf bytes.Buffer
			build := exec.Command("go", "build", "-o", bin, ".")
			build.Dir = outDir
			build.Stderr = &errBuf
			if err := build.Run(); err != nil {
				t.Fatalf("go build: %v\nstderr: %s", err, errBuf.String())
			}
			var out bytes.Buffer
			run := exec.Command(bin)
			run.Stdout = &out
			if err := run.Run(); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != wantPy {
				t.Fatalf("output mismatch\n--- python ---\n%s--- go ---\n%s", wantPy, got)
			}
		})
	}
}

// runPythonInDir runs `python3 <script>` from a working dir so sibling
// modules import correctly without needing PYTHONPATH gymnastics.
func runPythonInDir(t *testing.T, dir, script string) string {
	t.Helper()
	var out, errBuf bytes.Buffer
	cmd := exec.Command("python3", script)
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("python3 %s in %s: %v: %s", script, dir, err, errBuf.String())
	}
	return out.String()
}

// TestFixtures runs each fixture through Python and through the transpiler+Go
// build, and asserts the stdout matches. This is the only conformance gate.
func TestFixtures(t *testing.T) {
	root := repoRoot(t)
	fixDir := filepath.Join(root, "tests", "fixtures")
	entries, err := os.ReadDir(fixDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			pyPath := filepath.Join(fixDir, name)
			wantPy := runPython(t, pyPath)
			src := transpileFile(t, root, pyPath)
			gotGo := buildAndRunGo(t, src)
			if wantPy != gotGo {
				t.Fatalf("output mismatch\n--- python (%q) ---\n%s--- go (%q) ---\n%s--- generated source ---\n%s",
					wantPy, wantPy, gotGo, gotGo, src)
			}
		})
	}
}
