package transpile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rroblf01/gopy/ir"
	"github.com/rroblf01/gopy/parser"
)

// TestAsyncForLowering checks that `async for x in xs:` lowers as a regular
// for loop. Unlike CPython, gopy permits async-for over any iterable since
// it strips the async layer at lowering time (no real coroutine machinery
// underneath). The generated Go should range over the source like a sync
// for would.
func TestAsyncForLowering(t *testing.T) {
	src := `
async def total(xs):
    s = 0
    async for v in xs:
        s += v
    return s
`
	_ = src // type-hinted variant emitted to disk below
	srcTyped := "async def total(xs: list[int]) -> int:\n" +
		"    s: int = 0\n" +
		"    async for v in xs:\n" +
		"        s += v\n" +
		"    return s\n"

	tmp := t.TempDir()
	pyPath := filepath.Join(tmp, "async_for_inline.py")
	if err := os.WriteFile(pyPath, []byte(srcTyped), 0o644); err != nil {
		t.Fatal(err)
	}
	_, file, _, _ := runtime.Caller(0)
	dumper := filepath.Join(filepath.Dir(file), "..", "scripts", "py_ast_dump.py")

	node, err := parser.ParseFile(dumper, pyPath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	mod, err := ir.Lower(filepath.Base(pyPath), node)
	if err != nil {
		t.Fatalf("lower: %v", err)
	}
	out, err := Module(mod, Options{PackageName: "main"})
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "for _, v := range xs") {
		t.Fatalf("async-for not lowered as range; got:\n%s", got)
	}
}
