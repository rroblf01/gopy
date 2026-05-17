// Command gopy-build transpiles every .py file in a directory into a sibling
// .go file, all sharing the same Go package. Cross-module references work
// because Python identifiers in different files map onto the same Go package
// namespace; `from foo import bar` is dropped at lowering time.
//
// Usage:
//
//	gopy-build [-pkg main] [-o outdir] srcdir
//
// If -o is omitted, .go files are written next to their .py sources.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rroblf01/gopy/ir"
	"github.com/rroblf01/gopy/parser"
	"github.com/rroblf01/gopy/transpile"
)

func main() {
	pkg := flag.String("pkg", "main", "Go package name")
	outDir := flag.String("o", "", "output directory (default: alongside sources)")
	dumper := flag.String("dumper", "", "path to scripts/py_ast_dump.py (default: auto-locate)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gopy-build [-pkg name] [-o outdir] srcdir")
		os.Exit(2)
	}
	srcDir := flag.Arg(0)
	info, err := os.Stat(srcDir)
	if err != nil {
		die(err.Error())
	}
	if !info.IsDir() {
		die("srcdir must be a directory")
	}

	dumperPath := *dumper
	if dumperPath == "" {
		dumperPath = locateDumper()
		if dumperPath == "" {
			die("cannot locate py_ast_dump.py (pass -dumper)")
		}
	}

	dest := *outDir
	if dest == "" {
		dest = srcDir
	} else {
		check(os.MkdirAll(dest, 0o755))
	}

	entries, err := os.ReadDir(srcDir)
	check(err)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
			continue
		}
		pyPath := filepath.Join(srcDir, e.Name())
		root, err := parser.ParseFile(dumperPath, pyPath)
		if err != nil {
			die(fmt.Sprintf("%s: %v", pyPath, err))
		}
		mod, err := ir.Lower(e.Name(), root)
		if err != nil {
			die(fmt.Sprintf("%s: %v", pyPath, err))
		}
		src, err := transpile.Module(mod, transpile.Options{PackageName: *pkg})
		if err != nil {
			die(fmt.Sprintf("%s: %v", pyPath, err))
		}
		outFile := filepath.Join(dest, strings.TrimSuffix(e.Name(), ".py")+".go")
		check(os.WriteFile(outFile, src, 0o644))
		fmt.Fprintf(os.Stderr, "wrote %s\n", outFile)
	}
}

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
	fmt.Fprintln(os.Stderr, "gopy-build:", msg)
	os.Exit(1)
}
