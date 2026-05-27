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
	goformat "go/format"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	bridgepkg "github.com/rroblf01/gopy/bridge"
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
	bridge := fs.Bool("bridge", false, "enable the embedded-CPython bridge for non-stdlib imports (requires CGO + libpython)")
	goweb := fs.Bool("goweb", false, "map a recognized web framework (FastAPI/Flask) app onto a pure-Go net/http server")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gopy build [-o binary] [-keep] [-bridge] [-goweb] <input.py|dir>")
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
	info, err := os.Stat(src)
	check(err)
	if info.IsDir() {
		buildProject(src, *out, *python, *dumper, *keep)
		return
	}

	binPath := *out
	if binPath == "" {
		base := filepath.Base(src)
		binPath = strings.TrimSuffix(base, filepath.Ext(base))
	}
	absBin, err := filepath.Abs(binPath)
	check(err)

	goSrc, usedBridge := transpileFileBridge(src, "main", *python, *dumper, *bridge, *goweb)

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
	// When the program calls into the embedded interpreter, vendor the bridge
	// implementation as package main and build with CGO enabled so it links
	// libpython. Without bridged calls the binary stays pure Go.
	if usedBridge {
		check(os.WriteFile(filepath.Join(tmp, "gopy_bridge.go"), []byte(bridgepkg.MainPackageSource()), 0o644))
		check(os.WriteFile(filepath.Join(tmp, "gopy_bridge_reverse.go"), []byte(bridgepkg.ReverseSource()), 0o644))
		check(os.WriteFile(filepath.Join(tmp, "gopy_bridge_introspect.go"), []byte(bridgepkg.IntrospectSource()), 0o644))
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	}
	if err := cmd.Run(); err != nil {
		die("go build failed: " + err.Error())
	}
}

// buildProject handles `gopy build <dir>`: transpiles every .py in the dir
// into a sibling .go inside a staging tree, drops a minimal go.mod, then
// runs `go build`. The binary name is taken from -o, otherwise from
// pyproject.toml's `[project] name = "..."` line, otherwise from the
// directory's base name. requirements.txt is intentionally ignored —
// gopy doesn't resolve PyPI deps; libraries must be vendored as .py
// files alongside the application.
func buildProject(srcDir, outFlag, python, dumper string, keep bool) {
	binPath := outFlag
	if binPath == "" {
		binPath = detectProjectName(srcDir)
	}
	if binPath == "" {
		binPath = filepath.Base(filepath.Clean(srcDir))
	}
	absBin, err := filepath.Abs(binPath)
	check(err)
	dumperPath := dumper
	if dumperPath == "" {
		dumperPath = locateDumper()
		if dumperPath == "" {
			die("cannot locate py_ast_dump.py (pass -dumper)")
		}
	}
	pyBin := python
	if pyBin == "" {
		pyBin = parser.LocatePython(srcDir)
	}

	tmp, err := os.MkdirTemp("", "gopy-build-proj-")
	check(err)
	if !keep {
		defer os.RemoveAll(tmp)
	} else {
		fmt.Fprintf(os.Stderr, "gopy: keeping intermediate dir %s\n", tmp)
	}
	check(os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module gopy-build\n\ngo 1.22\n"), 0o644))
	entries, err := os.ReadDir(srcDir)
	check(err)
	wrote := 0
	sharedHelpers := map[string]string{}
	sharedImports := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
			continue
		}
		pyPath := filepath.Join(srcDir, e.Name())
		root, err := parser.ParseFileWith(pyBin, dumperPath, pyPath)
		if err != nil {
			die(fmt.Sprintf("%s: %v", pyPath, err))
		}
		mod, err := ir.Lower(e.Name(), root)
		if err != nil {
			die(fmt.Sprintf("%s: %v", pyPath, err))
		}
		src, meta, err := transpile.ModuleWithMeta(mod, transpile.Options{
			PackageName:  "main",
			SkipHelpers:  true,
			SourceModule: e.Name(),
		})
		if err != nil {
			die(fmt.Sprintf("%s: %v", pyPath, err))
		}
		for k, v := range meta.Helpers {
			sharedHelpers[k] = v
		}
		for _, imp := range meta.Imports {
			sharedImports[imp] = true
		}
		outFile := filepath.Join(tmp, strings.TrimSuffix(e.Name(), ".py")+".go")
		check(os.WriteFile(outFile, src, 0o644))
		wrote++
	}
	if wrote == 0 {
		die("no .py files found in " + srcDir)
	}
	if len(sharedHelpers) > 0 {
		check(os.WriteFile(filepath.Join(tmp, "gopy_runtime.go"), renderProjectRuntime("main", sharedHelpers, sharedImports), 0o644))
	}
	cmd := exec.Command("go", "build", "-o", absBin, ".")
	cmd.Dir = tmp
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		die("go build failed: " + err.Error())
	}
}

// renderProjectRuntime emits a shared gopy_runtime.go containing every
// inline helper any module in the project uses, plus the imports those
// helpers require. Without this consolidation each per-file translation
// would emit its own copy of __gopy_print / __gopy_repr / etc. and the
// linker would reject the package with "redeclared in this block".
func renderProjectRuntime(pkg string, helpers map[string]string, importSet map[string]bool) []byte {
	keys := make([]string, 0, len(helpers))
	for k := range helpers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	imps := make([]string, 0, len(importSet))
	for k := range importSet {
		imps = append(imps, k)
	}
	sort.Strings(imps)

	var b strings.Builder
	b.WriteString("// Code generated by gopy. DO NOT EDIT.\n\n")
	b.WriteString("package ")
	b.WriteString(pkg)
	b.WriteString("\n\n")
	if len(imps) > 0 {
		b.WriteString("import (\n")
		for _, i := range imps {
			b.WriteString("\t\"")
			b.WriteString(i)
			b.WriteString("\"\n")
		}
		b.WriteString(")\n\n")
	}
	for _, k := range keys {
		b.WriteString(helpers[k])
		b.WriteString("\n\n")
	}
	out := []byte(b.String())
	formatted, err := goformat.Source(out)
	if err != nil {
		return out
	}
	return formatted
}

// detectProjectName parses pyproject.toml shallowly for a `[project]`
// section's `name = "..."` line. Returns empty string when missing or
// malformed; caller falls back to the directory's basename. We avoid
// pulling a TOML parser in by handling only this single common shape.
func detectProjectName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	inProject := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") {
			inProject = line == "[project]"
			continue
		}
		if !inProject {
			continue
		}
		if !strings.HasPrefix(line, "name") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, `"'`)
		if val != "" {
			return val
		}
	}
	return ""
}

func transpileFile(src, pkg, python, dumper string) []byte {
	goSrc, _ := transpileFileBridge(src, pkg, python, dumper, false, false)
	return goSrc
}

// transpileFileBridge transpiles a single file, optionally enabling the
// embedded-CPython bridge. It returns the Go source and whether any bridged
// call was emitted (so the build driver knows to vendor the bridge + CGO).
func transpileFileBridge(src, pkg, python, dumper string, enableBridge, goWeb bool) ([]byte, bool) {
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
	goSrc, meta, err := transpile.ModuleWithMeta(mod, transpile.Options{
		PackageName:  pkg,
		SourceModule: modName,
		EnableBridge: enableBridge,
		GoWeb:        goWeb,
	})
	check(err)
	return goSrc, meta.UsedBridge
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
