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

// MainPackageSource returns the bridge implementation rewritten so it can be
// dropped into a generated `package main` build directory: the `package
// bridge` clause becomes `package main` and the `//go:build cgo` constraint
// line (plus its blank trailer) is removed, since the generated build always
// runs with CGO enabled. The resulting file defines Init / Import / Object and
// the conversion helpers directly in the program's package, so generated code
// calls them unqualified.
func MainPackageSource() string {
	s := bridgeSource
	// Drop the leading `//go:build cgo` line and the blank line after it.
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "//go:build cgo" {
			continue
		}
		out = append(out, ln)
	}
	s = strings.Join(out, "\n")
	s = strings.Replace(s, "\npackage bridge\n", "\npackage main\n", 1)
	return s
}
