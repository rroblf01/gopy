// This file has no build constraint so it compiles into the gopy tool even
// when CGO is disabled: it only embeds the bridge source as text and rewrites
// it for vendoring. The gopy tool itself never links libpython — only the
// generated binary does, when `gopy build` vendors this source into the build
// directory and compiles with CGO enabled.

package bridge

import (
	_ "embed"
	"strings"
)

//go:embed bridge.go
var bridgeSource string

//go:embed reverse.go
var reverseSource string

//go:embed introspect.go
var introspectSource string

// rewriteForMain strips the `//go:build cgo` constraint line and rewrites the
// `package bridge` clause to `package main`, so a bridge source file can be
// dropped into a generated build directory and compiled in-package.
func rewriteForMain(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "//go:build cgo" {
			continue
		}
		out = append(out, ln)
	}
	s = strings.Join(out, "\n")
	return strings.Replace(s, "\npackage bridge\n", "\npackage main\n", 1)
}

// MainPackageSource returns the forward-bridge implementation (Init / Import /
// Object plus the conversion helpers) rewritten for a generated `package main`
// build directory, so generated code calls them unqualified.
func MainPackageSource() string { return rewriteForMain(bridgeSource) }

// ReverseSource returns the reverse-bridge implementation (RegisterFunc and the
// C trampoline) rewritten for a generated `package main` build. Needed when the
// program exposes Go callbacks to Python (e.g. bridged decorators / routes).
func ReverseSource() string { return rewriteForMain(reverseSource) }

// IntrospectSource returns the introspection surface (RegisterTypedFunc,
// MakeClass, Param/Field) rewritten for a generated `package main` build, so a
// framework can introspect Go handlers as if they were native Python functions.
func IntrospectSource() string { return rewriteForMain(introspectSource) }
