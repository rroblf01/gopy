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
	venvDeps := fs.Bool("venv-deps", false, "resolve imports against the Python interpreter's site-packages too, recursively transpiling pure-Python dependencies from the venv (combine with -bridge to fall back to the interpreter for ones outside the supported subset)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gopy build [-o binary] [-keep] [-bridge] [-goweb] [-venv-deps] <input.py|dir>")
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

	// Resolve the toolchain paths once so import-graph discovery and per-module
	// transpilation agree on which Python / dumper to use.
	dumperPath := *dumper
	if dumperPath == "" {
		dumperPath = locateDumper()
		if dumperPath == "" {
			die("cannot locate py_ast_dump.py (pass -dumper)")
		}
	}
	pyBin := *python
	if pyBin == "" {
		pyBin = parser.LocatePython(filepath.Dir(src))
	}
	// Follow the import graph to sibling .py modules so a single-file build
	// recursively transpiles the local pure-Python deps it imports. With
	// -venv-deps, the interpreter's site-packages are added as extra resolution
	// roots so installed pure-Python libraries are transpiled too.
	var extraRoots []string
	if *venvDeps {
		extraRoots = venvSitePackages(pyBin)
		if len(extraRoots) == 0 {
			fmt.Fprintln(os.Stderr, "gopy: -venv-deps set but no site-packages found for the interpreter; resolving project-local modules only")
		}
	}
	deps, localMods := discoverLocalModules(src, pyBin, dumperPath, extraRoots)

	tmp, err := os.MkdirTemp("", "gopy-build-")
	check(err)
	if !*keep {
		defer os.RemoveAll(tmp)
	} else {
		fmt.Fprintf(os.Stderr, "gopy: keeping intermediate dir %s\n", tmp)
	}
	check(os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module gopy-build\n\ngo 1.22\n"), 0o644))

	if len(deps) == 0 {
		// No local deps: single self-contained main.go (helpers inline).
		goSrc, meta := transpileModule(src, "main", pyBin, dumperPath, *bridge, *goweb, false, nil)
		check(os.WriteFile(filepath.Join(tmp, "main.go"), goSrc, 0o644))
		runGoBuild(tmp, absBin, meta.UsedBridge)
		return
	}

	// Symbol-mangling metadata. Each module's top-level free functions and
	// module vars are mangled `<modpath dots→_>_<name>` so the flat shared Go
	// package has no collisions; classes stay bare. Compute every module's
	// dotted path, the set of all such paths, and each module's classes (so an
	// imported class name isn't mangled).
	entryDir := filepath.Dir(src)
	// rootOf maps each module file to the directory its dotted path is relative
	// to: the entry's dir for the entry and project-local modules, or a venv
	// site-packages dir for -venv-deps dependencies.
	rootOf := map[string]string{src: entryDir}
	for _, d := range deps {
		rootOf[d.path] = d.root
	}
	allFiles := append([]string{src}, func() []string {
		fs := make([]string, len(deps))
		for i, d := range deps {
			fs[i] = d.path
		}
		return fs
	}()...)
	pathOf := map[string]string{}
	localPathSet := map[string]bool{}
	var localModulePaths []string
	classesOf := map[string]map[string]bool{}
	for _, f := range allFiles {
		p := modulePathOf(f, src, rootOf[f])
		pathOf[f] = p
		if p != "" {
			localPathSet[p] = true
			localModulePaths = append(localModulePaths, p)
		}
		if root, err := parser.ParseFileWith(pyBin, dumperPath, f); err == nil {
			classesOf[p] = scanClasses(root)
		}
	}
	sort.Strings(localModulePaths)
	// Map each package __init__'s re-exports to the symbol's true defining
	// module, so `from pkg import name` resolves through the re-export chain.
	reExports := buildReExports(allFiles, pathOf, localPathSet, pyBin, dumperPath)
	// Collect cross-module top-level function signatures so a caller can
	// match kwargs / defaults against the callee's parameter list. Each dep
	// is lowered once; failures are silently skipped (the file may not be in
	// the supported subset — its sigs just won't be available for kwarg
	// resolution, and a kwarg call against it will fall back to the
	// existing error path).
	localFuncs := map[string]*ir.Func{}
	// loweredOK tracks which discovered files lowered cleanly. A dep that
	// fails (parse panic / unsupported subset feature) is DEMOTED from
	// "local" so its imports route through the embedded-CPython bridge
	// (under -bridge) instead of collapsing the whole build into Phase 2.
	// The survivors still transpile to Go; only the broken deps are bridged.
	loweredOK := map[string]bool{}
	for _, f := range allFiles {
		root, err := parser.ParseFileWith(pyBin, dumperPath, f)
		if err != nil {
			continue
		}
		// Lower with a panic guard: a malformed/unsupported dep (lowerFunc
		// has rough edges on heavy-dynamic libs like fastapi/pydantic) must
		// not kill the build — its signatures just won't be available for
		// kwarg resolution, and any call against it falls back to the
		// existing error / bridge path.
		var mod *ir.Module
		func() {
			defer func() { recover() }()
			mod, err = ir.Lower(filepath.Base(f), root)
		}()
		if err != nil || mod == nil {
			if os.Getenv("GOPY_TRACE_LOWER") != "" {
				fmt.Fprintf(os.Stderr, "gopy: lower FAIL %s (path=%s)\n", filepath.Base(f), pathOf[f])
			}
			continue
		}
		loweredOK[f] = true
		modPath := pathOf[f]
		for _, d := range mod.Decls {
			fn, ok := d.(*ir.Func)
			if !ok || fn.Receiver != nil {
				continue
			}
			mangled := mangleSym(modPath, fn.Name)
			// Shallow copy so the rewritten Name doesn't corrupt the module's
			// own IR if it's lowered again during transpile.
			sig := *fn
			sig.Name = mangled
			localFuncs[mangled] = &sig
		}
	}
	// If any dep failed to lower, prune the "local" classification so its
	// imports flip to bridged in transpile.buildAliases. Survivors stay
	// local and transpile normally. The entry must be in loweredOK — a
	// failed entry has no fallback (the entry is the program).
	if len(loweredOK) < len(allFiles) && *bridge {
		// Cascade demotion: a package whose `__init__.py` lowers fine but
		// whose submodule failed has a broken API (re-exports point at
		// nothing). Demote any ancestor package of a failed dep so the
		// whole subtree routes through the bridge — the user's call to
		// `uvicorn.run(...)` then hits the bridge with kwargs intact instead
		// of resolving to a half-translated `uvicorn_main_run`.
		failedAncestors := map[string]bool{}
		for _, f := range allFiles {
			if loweredOK[f] {
				continue
			}
			p := pathOf[f]
			for p != "" {
				failedAncestors[p] = true
				if idx := strings.LastIndex(p, "."); idx >= 0 {
					p = p[:idx]
				} else {
					p = ""
				}
			}
		}
		// Topological caveat: a survivor dep that imports a pruned dep will
		// still try to transpile, but its `import <pruned>` is now bridged
		// (good). A survivor that consumes the pruned dep's TYPES through
		// the codegen path may still error; that error is the existing
		// subset-gap path, not a regression.
		bridgedDeps := 0
		newDeps := deps[:0:len(deps)]
		newLocalPathSet := map[string]bool{}
		newLocalModulePaths := localModulePaths[:0]
		for _, d := range deps {
			p := pathOf[d.path]
			if loweredOK[d.path] && !failedAncestors[p] {
				newDeps = append(newDeps, d)
				if p != "" {
					newLocalPathSet[p] = true
					newLocalModulePaths = append(newLocalModulePaths, p)
				}
			} else {
				bridgedDeps++
			}
		}
		sort.Strings(newLocalModulePaths)
		// Rebuild localMods (qualifier-strip set) from the surviving deps.
		nameSet := map[string]bool{}
		for _, d := range newDeps {
			p := pathOf[d.path]
			parts := strings.Split(p, ".")
			nameSet[parts[len(parts)-1]] = true
			if len(parts) > 1 {
				nameSet[p] = true
			}
		}
		newLocalMods := make([]string, 0, len(nameSet))
		for n := range nameSet {
			newLocalMods = append(newLocalMods, n)
		}
		sort.Strings(newLocalMods)
		// Drop reExports whose source module was pruned (the re-export
		// target no longer transpiles, so an outside `from pkg import x`
		// must NOT rewrite to a non-existent mangled symbol).
		for pkg, names := range reExports {
			if !newLocalPathSet[pkg] {
				delete(reExports, pkg)
				continue
			}
			for n, r := range names {
				if !newLocalPathSet[r.fromMod] {
					delete(names, n)
				}
			}
		}
		fmt.Fprintf(os.Stderr, "gopy: %d venv dep(s) failed to lower; routing them through the bridge\n", bridgedDeps)
		deps = newDeps
		localPathSet = newLocalPathSet
		localModulePaths = newLocalModulePaths
		localMods = newLocalMods
	}

	// Recursive multi-file build. transpileDeps lists the discovered local
	// modules to transpile into the shared package; mangle enables symbol
	// mangling (Phase 1). Returns the go-build error (nil on success) without
	// exiting, so the caller can fall back to bridging.
	buildOnce := func(transpileDeps []localDep, mangle bool) error {
		// Clear any .go from a previous attempt (keep go.mod) so a stale dep
		// translation can't leak into a fallback build.
		if matches, _ := filepath.Glob(filepath.Join(tmp, "*.go")); matches != nil {
			for _, m := range matches {
				os.Remove(m)
			}
		}
		sharedHelpers := map[string]string{}
		sharedImports := map[string]bool{}
		usedBridge := false
		emit := func(path, goName string) error {
			opt := transpile.Options{EnableBridge: *bridge, GoWeb: *goweb, SkipHelpers: true, LocalModules: localMods}
			if mangle {
				opt.ModulePath = pathOf[path]
				opt.LocalModulePaths = localModulePaths
				if root, err := parser.ParseFileWith(pyBin, dumperPath, path); err == nil {
					opt.ImportedSymbols, opt.ModuleBindings = resolveImportBindings(root, packageDottedOf(path, rootOf[path]), localPathSet, classesOf, reExports)
				}
				if len(localFuncs) > 0 {
				opt.LocalFuncs = localFuncs
			}
			if len(reExports) > 0 {
					opt.LocalReExports = make(map[string]map[string]transpile.ReExport, len(reExports))
					for pkg, names := range reExports {
						if classesOf[pkg] == nil {
							classesOf[pkg] = map[string]bool{}
						}
						m := make(map[string]transpile.ReExport, len(names))
						for n, r := range names {
							// Skip class re-exports: classes stay bare per the
							// mangling design, so leave `pkg.Cls` un-rewritten.
							if classesOf[r.fromMod][r.origName] {
								continue
							}
							m[n] = transpile.ReExport{FromMod: r.fromMod, OrigName: r.origName}
						}
						opt.LocalReExports[pkg] = m
					}
				}
			}
			goSrc, meta, err := tryTranspileOpts(path, pyBin, dumperPath, opt)
			if err != nil {
				return fmt.Errorf("%s: %w", filepath.Base(path), err)
			}
			for k, v := range meta.Helpers {
				sharedHelpers[k] = v
			}
			for _, i := range meta.Imports {
				sharedImports[i] = true
			}
			if meta.UsedBridge {
				usedBridge = true
			}
			return os.WriteFile(filepath.Join(tmp, goName), goSrc, 0o644)
		}
		// The entry must transpile — it can't be bridged.
		if err := emit(src, "main.go"); err != nil {
			die(fmt.Sprintf("%s", err))
		}
		for _, d := range transpileDeps {
			if err := emit(d.path, d.goName); err != nil {
				return err
			}
		}
		if len(sharedHelpers) > 0 {
			check(os.WriteFile(filepath.Join(tmp, "gopy_runtime.go"), renderProjectRuntime("main", sharedHelpers, sharedImports), 0o644))
		}
		return runGoBuildErr(tmp, absBin, usedBridge)
	}

	// Phase 1: transpile the entry and every discovered module (pure Go), with
	// symbol mangling so same-named symbols across modules don't collide.
	err = buildOnce(deps, true)
	if err == nil {
		return
	}
	// Phase 2 (only with -bridge): a dep fell outside the supported subset
	// (untyped, dynamic features, …), so the pure build failed. Re-run with the
	// deps left to the embedded interpreter — only the entry is transpiled, and
	// `import dep` routes through the bridge. This is what keeps the transpiler
	// optimistic (transpile what fits) while still running the rest.
	if *bridge {
		fmt.Fprintf(os.Stderr, "gopy: pure recursion failed (%v); retrying with local deps bridged\n", err)
		if err2 := buildOnce(nil, false); err2 != nil {
			die("go build failed: " + err2.Error())
		}
		return
	}
	die(fmt.Sprintf("go build failed: %v (pass -bridge to run unsupported local deps in embedded CPython)", err))
}

// runGoBuild compiles the staged package in tmp into absBin, vendoring the
// bridge sources and enabling CGO when the program calls into embedded CPython.
// Exits on failure.
func runGoBuild(tmp, absBin string, usedBridge bool) {
	if err := runGoBuildErr(tmp, absBin, usedBridge); err != nil {
		die("go build failed: " + err.Error())
	}
}

// runGoBuildErr is runGoBuild that returns the error instead of exiting.
func runGoBuildErr(tmp, absBin string, usedBridge bool) error {
	cmd := exec.Command("go", "build", "-o", absBin, ".")
	cmd.Dir = tmp
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if usedBridge {
		check(os.WriteFile(filepath.Join(tmp, "gopy_bridge.go"), []byte(bridgepkg.MainPackageSource()), 0o644))
		check(os.WriteFile(filepath.Join(tmp, "gopy_bridge_reverse.go"), []byte(bridgepkg.ReverseSource()), 0o644))
		check(os.WriteFile(filepath.Join(tmp, "gopy_bridge_introspect.go"), []byte(bridgepkg.IntrospectSource()), 0o644))
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	}
	return cmd.Run()
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
	// Every .py stem is a sibling module in the shared package, so qualified
	// `mod.fn(...)` accesses drop the qualifier.
	var localMods []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".py") {
			localMods = append(localMods, strings.TrimSuffix(e.Name(), ".py"))
		}
	}
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
			LocalModules: localMods,
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
	goSrc, meta := transpileModule(src, pkg, python, dumper, enableBridge, goWeb, false, nil)
	return goSrc, meta.UsedBridge
}

// transpileModule parses, lowers, and transpiles one .py file. localMods names
// sibling modules transpiled into the same Go package (so `mod.fn(...)` drops
// the qualifier); skipHelpers omits inline runtime helpers so the caller can
// emit them once into a shared gopy_runtime.go for a multi-file build.
func transpileModule(src, pkg, python, dumper string, enableBridge, goWeb, skipHelpers bool, localMods []string) ([]byte, *transpile.ModuleMeta) {
	goSrc, meta, err := tryTranspileModule(src, pkg, python, dumper, enableBridge, goWeb, skipHelpers, localMods)
	check(err)
	return goSrc, meta
}

// tryTranspileModule is transpileModule that returns the error instead of
// exiting — used to probe whether a discovered local module is in the
// supported subset (and otherwise fall back to bridging it).
func tryTranspileModule(src, pkg, python, dumper string, enableBridge, goWeb, skipHelpers bool, localMods []string) ([]byte, *transpile.ModuleMeta, error) {
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
	if err != nil {
		return nil, nil, err
	}
	modName := filepath.Base(src)
	mod, err := ir.Lower(modName, root)
	if err != nil {
		return nil, nil, err
	}
	return transpile.ModuleWithMeta(mod, transpile.Options{
		PackageName:  pkg,
		SourceModule: modName,
		EnableBridge: enableBridge,
		GoWeb:        goWeb,
		SkipHelpers:  skipHelpers,
		LocalModules: localMods,
	})
}

// tryTranspileOpts parses + lowers a file and transpiles it with a
// caller-built Options (PackageName / SourceModule are filled in here). Used by
// the recursive build to pass per-module symbol-mangling configuration.
func tryTranspileOpts(src, python, dumper string, opt transpile.Options) ([]byte, *transpile.ModuleMeta, error) {
	root, err := parser.ParseFileWith(python, dumper, src)
	if err != nil {
		return nil, nil, err
	}
	modName := filepath.Base(src)
	mod, err := ir.Lower(modName, root)
	if err != nil {
		return nil, nil, err
	}
	opt.PackageName = "main"
	opt.SourceModule = modName
	return transpile.ModuleWithMeta(mod, opt)
}

// localDep is one discovered local dependency module: its source .py path and
// the unique .go filename it is emitted as (dotted module path → underscores,
// so `mypkg/calc.py` and a top-level `calc.py` don't collide on disk).
type localDep struct {
	path   string
	goName string
	// root is the directory the module's dotted import path is relative to —
	// the entry's dir for project-local modules, or a venv site-packages dir for
	// dependencies discovered there (`-venv-deps`).
	root string
}

// importRef is one candidate module an import statement might pull in. level is
// the relative-import depth (0 = absolute, 1 = `from .`, 2 = `from ..`, …);
// dotted is the module path to resolve, relative to the importing module's
// package when level > 0.
type importRef struct {
	level  int
	dotted string
}

// scanImports returns candidate module references from top-level `import X` /
// `from X import n1, n2` statements. For `import a.b` it yields "a.b"; for
// `from pkg import mod` it yields "pkg" (the name may live in pkg/__init__.py)
// and "pkg.mod" (it may be a submodule). Relative imports carry their level:
// `from .mod import y` → {1,"mod"},{1,"mod.y"}; `from . import x` → {1,"x"}.
func scanImports(root parser.Node) []importRef {
	var out []importRef
	for _, stmt := range root.Children("body") {
		switch stmt.Type() {
		case "Import":
			for _, a := range stmt.Children("names") {
				if nm := a.Str("name"); nm != "" {
					out = append(out, importRef{0, nm})
				}
			}
		case "ImportFrom":
			level := 0
			if lvl, ok := stmt["level"].(float64); ok {
				level = int(lvl)
			}
			m := stmt.Str("module")
			if m != "" {
				out = append(out, importRef{level, m})
			}
			for _, a := range stmt.Children("names") {
				nm := a.Str("name")
				if nm == "" {
					continue
				}
				if m != "" {
					out = append(out, importRef{level, m + "." + nm})
				} else if level > 0 {
					// `from . import nm` — nm is a submodule of the current package.
					out = append(out, importRef{level, nm})
				}
			}
		}
	}
	return out
}

// resolveModuleFile maps a dotted module path to a local .py file under baseDir,
// trying the module-file layout (`a/b/c.py`) then the package layout
// (`a/b/c/__init__.py`). Returns "" when neither exists (stdlib / third-party /
// not local).
func resolveModuleFile(dotted, baseDir string) string {
	parts := strings.Split(dotted, ".")
	asModule := filepath.Join(append([]string{baseDir}, parts...)...) + ".py"
	if _, err := os.Stat(asModule); err == nil {
		return asModule
	}
	asPackage := filepath.Join(append([]string{baseDir}, append(parts, "__init__.py")...)...)
	if _, err := os.Stat(asPackage); err == nil {
		return asPackage
	}
	return ""
}

// discoverLocalModules walks the import graph from entryPath, following
// `import X` / `from X import ...` to local .py files under the entry's
// directory — siblings (`utils.py`) and subpackages alike (`pkg/sub.py`,
// `pkg/__init__.py`). It returns the dependency modules (excluding the entry)
// in discovery order and the set of local module names used for qualifier
// stripping. Imports that don't resolve to a local file (stdlib / third-party)
// are left alone. Flat-package caveat: every module shares one Go package, so
// two modules defining the same top-level symbol collide — keep names unique
// across the project (or that build falls back to the bridge under -bridge).
// venvSitePackages asks the interpreter for its site-packages directories
// (purelib + platlib). Under -venv-deps these become extra module-resolution
// roots, so installed pure-Python dependencies are discovered and transpiled
// alongside the application.
func venvSitePackages(pyBin string) []string {
	out, err := exec.Command(pyBin, "-c",
		"import sysconfig; print(sysconfig.get_path('purelib')); print(sysconfig.get_path('platlib'))").Output()
	if err != nil {
		return nil
	}
	var roots []string
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		p := strings.TrimSpace(line)
		if p == "" || seen[p] {
			continue
		}
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			seen[p] = true
			roots = append(roots, p)
		}
	}
	return roots
}

func discoverLocalModules(entryPath, pyBin, dumperPath string, extraRoots []string) (deps []localDep, modNames []string) {
	dir := filepath.Dir(entryPath)
	entryAbs, _ := filepath.Abs(entryPath)
	seen := map[string]bool{entryAbs: true}
	nameSet := map[string]bool{}
	// Each queued file remembers the root its dotted path is relative to, so a
	// dependency pulled from a venv site-packages dir (and its own relative
	// imports) resolve and name themselves against that root, not the entry's.
	type work struct{ file, root string }
	queue := []work{{entryAbs, dir}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		root, err := parser.ParseFileWith(pyBin, dumperPath, cur.file)
		if err != nil {
			continue // unparseable: don't discover through it
		}
		for _, ref := range scanImports(root) {
			// Absolute imports resolve from the project root (the entry's dir),
			// then any extra roots (venv site-packages); relative imports resolve
			// from the importing module's package, ascending one level per
			// leading dot beyond the first, within that module's own root.
			cand, candRoot := "", ""
			if ref.level > 0 {
				base := filepath.Dir(cur.file)
				for i := 1; i < ref.level; i++ {
					base = filepath.Dir(base)
				}
				cand, candRoot = resolveModuleFile(ref.dotted, base), cur.root
			} else {
				for _, r := range append([]string{dir}, extraRoots...) {
					if c := resolveModuleFile(ref.dotted, r); c != "" {
						cand, candRoot = c, r
						break
					}
				}
			}
			if cand == "" {
				continue // not a local / venv module
			}
			// Register both the last dotted segment (the bare binding a
			// sibling-style `from pkg import mod; mod.fn()` references) and the
			// full dotted path (what `import pkg.mod; pkg.mod.fn()` references)
			// for qualifier stripping.
			parts := strings.Split(ref.dotted, ".")
			nameSet[parts[len(parts)-1]] = true
			if len(parts) > 1 {
				nameSet[ref.dotted] = true
			}
			abs, _ := filepath.Abs(cand)
			if !seen[abs] {
				seen[abs] = true
				queue = append(queue, work{abs, candRoot})
				rel, err := filepath.Rel(candRoot, abs)
				if err != nil {
					rel = filepath.Base(abs)
				}
				goName := strings.NewReplacer("/", "_", string(filepath.Separator), "_", ".py", "").Replace(rel) + ".go"
				deps = append(deps, localDep{path: cand, goName: goName, root: candRoot})
			}
		}
	}
	for n := range nameSet {
		modNames = append(modNames, n)
	}
	sort.Strings(modNames)
	return deps, modNames
}

// modulePathOf returns a module file's dotted import path relative to the entry
// directory ("" for the entry itself or a top-level __init__.py). E.g.
// mypkg/calc.py → "mypkg.calc", mypkg/__init__.py → "mypkg".
func modulePathOf(file, entry, entryDir string) string {
	abs, _ := filepath.Abs(file)
	if ea, _ := filepath.Abs(entry); abs == ea {
		return ""
	}
	rel, err := filepath.Rel(entryDir, abs)
	if err != nil {
		return ""
	}
	rel = strings.TrimSuffix(rel, ".py")
	rel = strings.TrimSuffix(rel, string(filepath.Separator)+"__init__")
	if rel == "__init__" {
		return ""
	}
	return strings.ReplaceAll(rel, string(filepath.Separator), ".")
}

// packageDottedOf returns the dotted path of a module file's *package* (its
// directory relative to the entry dir), against which relative imports resolve.
func packageDottedOf(file, entryDir string) string {
	rel, err := filepath.Rel(entryDir, filepath.Dir(file))
	if err != nil || rel == "." {
		return ""
	}
	return strings.ReplaceAll(rel, string(filepath.Separator), ".")
}

// mangleSym is the driver-side mirror of transpile's symbol mangling: a module
// path (dots → underscores) joined to a symbol name. Must stay in sync with
// transpile's gen.mangle so cross-module references resolve to the same symbol.
func mangleSym(path, name string) string {
	if path == "" {
		return name
	}
	return strings.ReplaceAll(path, ".", "_") + "_" + name
}

// scanClasses returns the names of top-level classes defined in a parsed module
// — used so an imported class name is left un-mangled (classes stay bare).
func scanClasses(root parser.Node) map[string]bool {
	out := map[string]bool{}
	for _, s := range root.Children("body") {
		if s.Type() == "ClassDef" {
			if n := s.Str("name"); n != "" {
				out[n] = true
			}
		}
	}
	return out
}

// reExport records that a package's `__init__.py` rebinds an outside name from
// some submodule: `from .core import add as plus` in `mylib/__init__.py` gives
// reExports["mylib"]["plus"] = {fromMod: "mylib.core", origName: "add"}.
type reExport struct {
	fromMod  string
	origName string
}

// buildReExports scans every package `__init__.py` in allFiles for
// `from <mod> import name [as alias]` lines and records each rebound name's
// source module + original name, so an outside `from pkg import name` resolves
// to the symbol's true defining module. Chains are collapsed to a fixed point,
// so `a.__init__` re-exporting through `b.__init__` lands on b's defining
// module directly. Only references to local modules (per localPath) are
// recorded — stdlib / venv / third-party re-exports are left alone.
func buildReExports(allFiles []string, pathOf map[string]string, localPath map[string]bool, pyBin, dumperPath string) map[string]map[string]reExport {
	re := map[string]map[string]reExport{}
	resolveMod := func(pkgPath string, level int, module string) string {
		if level == 0 {
			return module
		}
		base := pkgPath
		for i := 1; i < level; i++ {
			if idx := strings.LastIndex(base, "."); idx >= 0 {
				base = base[:idx]
			} else {
				base = ""
			}
		}
		switch {
		case module == "":
			return base
		case base == "":
			return module
		default:
			return base + "." + module
		}
	}
	for _, f := range allFiles {
		if filepath.Base(f) != "__init__.py" {
			continue
		}
		pkgPath := pathOf[f]
		if pkgPath == "" {
			continue
		}
		root, err := parser.ParseFileWith(pyBin, dumperPath, f)
		if err != nil {
			continue
		}
		for _, s := range root.Children("body") {
			if s.Type() != "ImportFrom" {
				continue
			}
			level := 0
			if lv, ok := s["level"].(float64); ok {
				level = int(lv)
			}
			fm := resolveMod(pkgPath, level, s.Str("module"))
			if !localPath[fm] {
				continue
			}
			for _, a := range s.Children("names") {
				name := a.Str("name")
				if name == "" || name == "*" {
					continue
				}
				bind := a.Str("asname")
				if bind == "" {
					bind = name
				}
				if re[pkgPath] == nil {
					re[pkgPath] = map[string]reExport{}
				}
				re[pkgPath][bind] = reExport{fromMod: fm, origName: name}
			}
		}
	}
	// Collapse transitive chains: if pkg.x → (m, y) and m.y → (n, z), then
	// pkg.x → (n, z). Iterate to a fixed point.
	for {
		changed := false
		for pkg, names := range re {
			for n, r := range names {
				if next, ok := re[r.fromMod]; ok {
					if nr, ok2 := next[r.origName]; ok2 {
						re[pkg][n] = nr
						changed = true
					}
				}
			}
		}
		if !changed {
			break
		}
	}
	return re
}

// resolveImportBindings inspects a module's import statements and classifies
// each imported name for symbol mangling: a submodule binding (`from pkg import
// sub`, `import M`) → modBindings[name]=path; a value symbol (`from M import
// fn`) → imported[name]=M; a class → skipped (classes stay bare). Relative
// imports resolve against pkg (the importing module's package). localPath is
// the set of all local module dotted paths; classesOf maps a module path to its
// class names. reExports collapses package `__init__.py` re-exports so a value
// imported through a package resolves to its true defining module.
func resolveImportBindings(root parser.Node, pkg string, localPath map[string]bool, classesOf map[string]map[string]bool, reExports map[string]map[string]reExport) (imported, modBindings map[string]string) {
	imported = map[string]string{}
	modBindings = map[string]string{}
	resolveMod := func(level int, module string) string {
		if level == 0 {
			return module
		}
		base := pkg
		for i := 1; i < level; i++ {
			if idx := strings.LastIndex(base, "."); idx >= 0 {
				base = base[:idx]
			} else {
				base = ""
			}
		}
		switch {
		case module == "":
			return base
		case base == "":
			return module
		default:
			return base + "." + module
		}
	}
	for _, s := range root.Children("body") {
		switch s.Type() {
		case "Import":
			for _, a := range s.Children("names") {
				name := a.Str("name")
				if name == "" || !localPath[name] {
					continue
				}
				bind := a.Str("asname")
				if bind == "" {
					bind = name
				}
				modBindings[bind] = name
			}
		case "ImportFrom":
			level := 0
			if lv, ok := s["level"].(float64); ok {
				level = int(lv)
			}
			fm := resolveMod(level, s.Str("module"))
			if !localPath[fm] {
				continue
			}
			for _, a := range s.Children("names") {
				name := a.Str("name")
				if name == "" {
					continue
				}
				bind := a.Str("asname")
				if bind == "" {
					bind = name
				}
				// Follow a package __init__'s re-export chain so a value imported
				// through the package resolves to its true defining module
				// (`from mylib import add` where mylib/__init__.py does
				// `from .core import add` → src=mylib.core, orig=add).
				src, orig := fm, name
				if re, ok := reExports[fm]; ok {
					if r, ok2 := re[name]; ok2 {
						src, orig = r.fromMod, r.origName
					}
				}
				switch {
				case localPath[fm+"."+name]:
					modBindings[bind] = fm + "." + name
				case classesOf[src][orig]:
					// class — stays bare, no mangling binding
				default:
					// Value symbol: resolve the binding (possibly aliased) to the
					// source module's mangled symbol, e.g. `from a import f as g`
					// → g → "a_f".
					imported[bind] = mangleSym(src, orig)
				}
			}
		}
	}
	return imported, modBindings
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
