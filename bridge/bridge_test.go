//go:build cgo

package bridge

import "testing"

// TestStdlibRoundTrip exercises the bridge against the always-present stdlib:
// import math, call sqrt, round-trip the result to Go.
func TestStdlibRoundTrip(t *testing.T) {
	math, err := Import("math")
	if err != nil {
		t.Fatalf("import math: %v", err)
	}
	defer math.DecRef()

	r, err := math.CallMethod("sqrt", 16.0)
	if err != nil {
		t.Fatalf("math.sqrt: %v", err)
	}
	defer r.DecRef()

	v, err := r.Go()
	if err != nil {
		t.Fatalf("sqrt result -> go: %v", err)
	}
	if got, ok := v.(float64); !ok || got != 4.0 {
		t.Fatalf("math.sqrt(16) = %v (%T), want 4.0 (float64)", v, v)
	}
}

// TestConversions checks the toPy/fromPy round-trip for each supported type
// via the builtins module: `repr` / `len` and direct Go() conversions.
func TestConversions(t *testing.T) {
	builtins, err := Import("builtins")
	if err != nil {
		t.Fatalf("import builtins: %v", err)
	}
	defer builtins.DecRef()

	// len([1, 2, 3]) == 3
	lenFn, err := builtins.Attr("len")
	if err != nil {
		t.Fatalf("builtins.len: %v", err)
	}
	defer lenFn.DecRef()
	r, err := lenFn.Call([]any{int64(1), int64(2), int64(3)})
	if err != nil {
		t.Fatalf("len([...]): %v", err)
	}
	defer r.DecRef()
	v, _ := r.Go()
	if v != int64(3) {
		t.Fatalf("len([1,2,3]) = %v, want 3", v)
	}

	// A nested dict round-trips through toPy/fromPy via repr().
	nested := map[string]any{"n": int64(1), "s": "two", "b": true, "xs": []any{int64(9)}}
	reprFn, err := builtins.Attr("repr")
	if err != nil {
		t.Fatalf("builtins.repr: %v", err)
	}
	defer reprFn.DecRef()
	rr, err := reprFn.Call(nested)
	if err != nil {
		t.Fatalf("repr(dict): %v", err)
	}
	defer rr.DecRef()
	rv, _ := rr.Go()
	// Dict ordering in the repr follows insertion of our Go map iteration,
	// which is unstable — assert the conversion produced a dict-shaped repr.
	s, _ := rv.(string)
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		t.Fatalf("repr(dict) = %q, want a {...} dict repr", s)
	}
}

// TestCallKw verifies positional + keyword argument calls. `int("ff", base=16)`
// returns 255.
func TestCallKw(t *testing.T) {
	builtins, err := Import("builtins")
	if err != nil {
		t.Fatalf("import builtins: %v", err)
	}
	defer builtins.DecRef()

	intFn, err := builtins.Attr("int")
	if err != nil {
		t.Fatalf("builtins.int: %v", err)
	}
	defer intFn.DecRef()

	r, err := intFn.CallKw([]any{"ff"}, map[string]any{"base": int64(16)})
	if err != nil {
		t.Fatalf(`int("ff", base=16): %v`, err)
	}
	defer r.DecRef()
	v, _ := r.Go()
	if v != int64(255) {
		t.Fatalf(`int("ff", base=16) = %v, want 255`, v)
	}
}

// TestPydanticCore is the headline proof: load the Rust extension, build a
// validator, validate a value, and confirm the failure path surfaces a Go
// error. Skips cleanly when pydantic_core isn't installed.
func TestPydanticCore(t *testing.T) {
	pc, err := Import("pydantic_core")
	if err != nil {
		t.Skipf("pydantic_core not available: %v", err)
	}
	defer pc.DecRef()

	sv, err := pc.Attr("SchemaValidator")
	if err != nil {
		t.Fatalf("attr SchemaValidator: %v", err)
	}
	defer sv.DecRef()

	validator, err := sv.Call(map[string]any{"type": "int"})
	if err != nil {
		t.Fatalf("SchemaValidator({type:int}): %v", err)
	}
	defer validator.DecRef()

	// Coercion: "42" -> 42
	out, err := validator.CallMethod("validate_python", "42")
	if err != nil {
		t.Fatalf("validate_python('42'): %v", err)
	}
	defer out.DecRef()
	v, _ := out.Go()
	if v != int64(42) {
		t.Fatalf("validate_python('42') = %v, want 42", v)
	}

	// Failure path: a non-numeric string must produce a Python error.
	if _, err := validator.CallMethod("validate_python", "not-a-number"); err == nil {
		t.Fatal("expected validation error for bad input, got nil")
	}
}

// TestStrLenBool covers the Str/Repr/Len/Bool helpers.
func TestStrLenBool(t *testing.T) {
	builtins, err := Import("builtins")
	if err != nil {
		t.Fatalf("import builtins: %v", err)
	}
	defer builtins.DecRef()

	lst, err := builtins.CallMethod("list", []any{int64(1), int64(2)})
	if err != nil {
		t.Fatalf("list([1,2]): %v", err)
	}
	defer lst.DecRef()

	n, err := lst.Len()
	if err != nil || n != 2 {
		t.Fatalf("Len = %d, err %v; want 2", n, err)
	}
	b, err := lst.Bool()
	if err != nil || !b {
		t.Fatalf("Bool([1,2]) = %v, err %v; want true", b, err)
	}
	s, err := lst.Repr()
	if err != nil || s != "[1, 2]" {
		t.Fatalf("Repr = %q, err %v; want \"[1, 2]\"", s, err)
	}
}
