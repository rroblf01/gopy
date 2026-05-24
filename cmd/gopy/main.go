// Command gopy transpiles a Python source file to Go and (with the build
// subcommand) compiles it into a single statically-linked binary.
//
//	gopy [-o out.go] [-pkg main] input.py            # emit Go source
//	gopy build [-o binary] input.py                  # transpile + go build
//	gopy watch [-interval 500ms] [-o binary] in.py   # re-run build on change
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rroblf01/gopy/ir"
	"github.com/rroblf01/gopy/parser"
	"github.com/rroblf01/gopy/transpile"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "build":
			runBuild(os.Args[2:])
			return
		case "watch":
			runWatch(os.Args[2:])
			return
		}
	}
	runEmit(os.Args[1:])
}

// runEmit implements the original behavior: transpile a .py file to Go
// source, writing to -o or stdout.
func runEmit(args []string) {
	fs := flag.NewFlagSet("gopy", flag.ExitOnError)
	out := fs.String("o", "", "output Go file (default: stdout)")
	pkg := fs.String("pkg", "main", "Go package name for generated file")
	dumper := fs.String("dumper", "", "path to scripts/py_ast_dump.py (default: auto-locate)")
	python := fs.String("python", "", "Python interpreter to use (default: ./.venv/bin/python3 if present, else python3)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gopy [-o out.go] [-pkg name] input.py")
		fmt.Fprintln(os.Stderr, "       gopy build [-o binary] input.py")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	src := fs.Arg(0)
	goSrc := transpileFile(src, *pkg, *python, *dumper)
	if *out == "" {
		os.Stdout.Write(goSrc)
		return
	}
	check(os.WriteFile(*out, goSrc, 0o644))
}

// runWatch keeps `gopy build` warm: it polls the input .py's modification
// time and re-runs the build whenever it changes. Pure-poll (no fsnotify
// dependency) — fine for the rebuild cadence and avoids pulling third
// party deps into the module.
func runWatch(args []string) {
	fs := flag.NewFlagSet("gopy watch", flag.ExitOnError)
	out := fs.String("o", "", "output binary path (default: stem of input)")
	dumper := fs.String("dumper", "", "path to scripts/py_ast_dump.py (default: auto-locate)")
	python := fs.String("python", "", "Python interpreter to use")
	interval := fs.Duration("interval", 500*time.Millisecond, "poll interval")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gopy watch [-o binary] [-interval 500ms] input.py")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	src := fs.Arg(0)
	last := time.Time{}
	for {
		info, err := os.Stat(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gopy watch: %v\n", err)
			time.Sleep(*interval)
			continue
		}
		if info.ModTime() != last {
			last = info.ModTime()
			fmt.Fprintf(os.Stderr, "gopy watch: rebuilding %s\n", src)
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Fprintf(os.Stderr, "gopy watch: build failed: %v\n", r)
					}
				}()
				buildOnce(src, *out, *python, *dumper)
			}()
		}
		time.Sleep(*interval)
	}
}

// buildOnce performs a single transpile + go-build cycle. Errors during
// transpile / build print to stderr and return — the watch loop swallows
// the panic so a transient bad save doesn't kill the watcher.
func buildOnce(src, out, python, dumper string) {
	binPath := out
	if binPath == "" {
		base := filepath.Base(src)
		binPath = strings.TrimSuffix(base, filepath.Ext(base))
	}
	absBin, err := filepath.Abs(binPath)
	check(err)
	goSrc := transpileFile(src, "main", python, dumper)
	tmp, err := os.MkdirTemp("", "gopy-watch-")
	check(err)
	defer os.RemoveAll(tmp)
	goFile := filepath.Join(tmp, "main.go")
	check(os.WriteFile(goFile, goSrc, 0o644))
	check(os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module gopy-watch\n\ngo 1.22\n"), 0o644))
	cmd := exec.Command("go", "build", "-o", absBin, ".")
	cmd.Dir = tmp
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gopy watch: go build: %v\n", err)
	}
}

// runBuild transpiles the input .py and runs `go build` to produce a native
// binary in one step. The Go source is staged in a temp directory along with
// a minimal go.mod so the toolchain has a module to build.
func runBuild(args []string) {
	fs := flag.NewFlagSet("gopy build", flag.ExitOnError)
	out := fs.String("o", "", "output binary path (default: stem of input)")
	dumper := fs.String("dumper", "", "path to scripts/py_ast_dump.py (default: auto-locate)")
	python := fs.String("python", "", "Python interpreter to use (default: ./.venv/bin/python3 if present, else python3)")
	keep := fs.Bool("keep", false, "keep the intermediate Go source directory (printed to stderr)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gopy build [-o binary] [-keep] input.py")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	src := fs.Arg(0)
	binPath := *out
	if binPath == "" {
		base := filepath.Base(src)
		binPath = strings.TrimSuffix(base, filepath.Ext(base))
	}
	absBin, err := filepath.Abs(binPath)
	check(err)

	goSrc := transpileFile(src, "main", *python, *dumper)

	tmp, err := os.MkdirTemp("", "gopy-build-")
	check(err)
	if !*keep {
		defer os.RemoveAll(tmp)
	} else {
		fmt.Fprintf(os.Stderr, "gopy: keeping intermediate dir %s\n", tmp)
	}
	goFile := filepath.Join(tmp, "main.go")
	check(os.WriteFile(goFile, goSrc, 0o644))
	modContent := "module gopy-build\n\ngo 1.22\n"
	check(os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(modContent), 0o644))

	cmd := exec.Command("go", "build", "-o", absBin, ".")
	cmd.Dir = tmp
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		die("go build failed: " + err.Error())
	}
}

func transpileFile(src, pkg, python, dumper string) []byte {
	dumperPath := dumper
	if dumperPath == "" {
		dumperPath = locateDumper()
		if dumperPath == "" {
			die("cannot locate py_ast_dump.py (pass -dumper)")
		}
	}
	pyBin := python
	if pyBin == "" {
		pyBin = parser.LocatePython(src)
	}
	root, err := parser.ParseFileWith(pyBin, dumperPath, src)
	check(err)
	modName := filepath.Base(src)
	mod, err := ir.Lower(modName, root)
	check(err)
	goSrc, err := transpile.Module(mod, transpile.Options{PackageName: pkg})
	check(err)
	return goSrc
}

// locateDumper walks up from the caller's source dir, or from CWD, looking
// for scripts/py_ast_dump.py. Useful for development; production builds
// will pin -dumper explicitly.
func locateDumper() string {
	candidates := []string{}
	if _, file, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(file), "..", "..", "scripts", "py_ast_dump.py"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "scripts", "py_ast_dump.py"),
			filepath.Join(cwd, "..", "scripts", "py_ast_dump.py"),
		)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

func check(err error) {
	if err != nil {
		die(err.Error())
	}
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, "gopy:", msg)
	os.Exit(1)
}
