// Command gopy transpiles a Python source file to Go.
//
//	gopy [-o out.go] [-pkg main] input.py
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rroblf01/gopy/ir"
	"github.com/rroblf01/gopy/parser"
	"github.com/rroblf01/gopy/transpile"
)

func main() {
	out := flag.String("o", "", "output Go file (default: stdout)")
	pkg := flag.String("pkg", "main", "Go package name for generated file")
	dumper := flag.String("dumper", "", "path to scripts/py_ast_dump.py (default: auto-locate)")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gopy [-o out.go] [-pkg name] input.py")
		os.Exit(2)
	}
	src := flag.Arg(0)
	dumperPath := *dumper
	if dumperPath == "" {
		dumperPath = locateDumper()
		if dumperPath == "" {
			die("cannot locate py_ast_dump.py (pass -dumper)")
		}
	}

	root, err := parser.ParseFile(dumperPath, src)
	check(err)

	modName := filepath.Base(src)
	mod, err := ir.Lower(modName, root)
	check(err)

	goSrc, err := transpile.Module(mod, transpile.Options{PackageName: *pkg})
	check(err)

	if *out == "" {
		os.Stdout.Write(goSrc)
		return
	}
	check(os.WriteFile(*out, goSrc, 0o644))
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
