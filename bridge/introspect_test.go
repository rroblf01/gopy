//go:build cgo

package bridge

import "testing"

// TestTypedFuncSignature exposes a Go function to Python with a real
// signature and checks that inspect.signature + typing.get_type_hints report
// it correctly — the property frameworks like FastAPI depend on — and that
// the wrapper is callable, forwarding to the Go implementation.
func TestTypedFuncSignature(t *testing.T) {
	// def add(a: int, b: int = 5) -> int
	add, err := RegisterTypedFunc("add",
		[]Param{
			{Name: "a", Annotation: "int"},
			{Name: "b", Annotation: "int", HasDefault: true, Default: int64(5)},
		},
		"int",
		func(args []any, kwargs map[string]any) any {
			return args[0].(int64) + args[1].(int64)
		},
	)
	if err != nil {
		t.Fatalf("RegisterTypedFunc: %v", err)
	}
	defer add.DecRef()

	inspectMod, err := Import("inspect")
	if err != nil {
		t.Fatalf("import inspect: %v", err)
	}
	defer inspectMod.DecRef()

	// str(inspect.signature(add)) == "(a: int, b: int = 5) -> int"
	sig, err := inspectMod.CallMethod("signature", add)
	if err != nil {
		t.Fatalf("inspect.signature(add): %v", err)
	}
	defer sig.DecRef()
	sigStr, err := sig.Str()
	if err != nil {
		t.Fatalf("str(sig): %v", err)
	}
	if sigStr != "(a: int, b: int = 5) -> int" {
		t.Fatalf("signature = %q, want \"(a: int, b: int = 5) -> int\"", sigStr)
	}

	// typing.get_type_hints(add) should map a/b/return to int.
	typingMod, err := Import("typing")
	if err != nil {
		t.Fatalf("import typing: %v", err)
	}
	defer typingMod.DecRef()
	hints, err := typingMod.CallMethod("get_type_hints", add)
	if err != nil {
		t.Fatalf("get_type_hints: %v", err)
	}
	defer hints.DecRef()
	hlen, _ := hints.Len()
	if hlen != 3 {
		t.Fatalf("get_type_hints len = %d, want 3 (a, b, return)", hlen)
	}

	// Call through Python: add(10) uses the default b=5 → 15.
	r, err := add.Call(int64(10))
	if err != nil {
		t.Fatalf("add(10): %v", err)
	}
	defer r.DecRef()
	v, _ := r.Go()
	if v != int64(15) {
		t.Fatalf("add(10) = %v, want 15 (b defaults to 5)", v)
	}

	// add(3, 4) → 7
	r2, err := add.Call(int64(3), int64(4))
	if err != nil {
		t.Fatalf("add(3,4): %v", err)
	}
	defer r2.DecRef()
	v2, _ := r2.Go()
	if v2 != int64(7) {
		t.Fatalf("add(3,4) = %v, want 7", v2)
	}
}
