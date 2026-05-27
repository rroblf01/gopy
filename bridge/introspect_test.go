//go:build cgo

package bridge

import "testing"

// frameworkDispatchSrc is the exact pattern a web framework uses: introspect a
// handler's signature, coerce raw (string) request params to each parameter's
// annotated type, then call. Proves a transpiled Go handler integrates with a
// Python framework's request-binding machinery.
const frameworkDispatchSrc = `
import inspect

def _gopy_dispatch(handler, raw_params):
    sig = inspect.signature(handler)
    kwargs = {}
    for pname, param in sig.parameters.items():
        if pname in raw_params:
            val = raw_params[pname]
            ann = param.annotation
            if ann is int:
                val = int(val)
            elif ann is float:
                val = float(val)
            elif ann is bool:
                val = val in ('1', 'true', 'True', True)
            else:
                val = str(val)
            kwargs[pname] = val
    return handler(**kwargs)
`

// TestFrameworkDispatch proves the end-to-end framework integration: a Python
// dispatcher introspects a Go handler's signature, coerces string request
// params to the handler's annotated types, and invokes it — the core of what
// FastAPI does per request.
func TestFrameworkDispatch(t *testing.T) {
	// def add(a: int, b: int) -> int, backed by Go.
	add, err := RegisterTypedFunc("add",
		[]Param{{Name: "a", Annotation: "int"}, {Name: "b", Annotation: "int"}},
		"int",
		func(args []any, kwargs map[string]any) any {
			return args[0].(int64) + args[1].(int64)
		},
	)
	if err != nil {
		t.Fatalf("RegisterTypedFunc: %v", err)
	}
	defer add.DecRef()

	// Bootstrap the dispatcher.
	builtins, err := Import("builtins")
	if err != nil {
		t.Fatalf("import builtins: %v", err)
	}
	defer builtins.DecRef()
	ns, err := builtins.CallMethod("dict")
	if err != nil {
		t.Fatalf("dict(): %v", err)
	}
	defer ns.DecRef()
	if _, err := builtins.CallMethod("exec", frameworkDispatchSrc, ns); err != nil {
		t.Fatalf("exec dispatcher: %v", err)
	}
	dispatch, err := ns.GetItem("_gopy_dispatch")
	if err != nil {
		t.Fatalf("get _gopy_dispatch: %v", err)
	}
	defer dispatch.DecRef()

	// Simulate a request: raw string params, as an HTTP layer would deliver.
	r, err := dispatch.Call(add, map[string]any{"a": "3", "b": "4"})
	if err != nil {
		t.Fatalf("dispatch(add, {a:3,b:4}): %v", err)
	}
	defer r.DecRef()
	v, _ := r.Go()
	if v != int64(7) {
		t.Fatalf("dispatch coerced+called add(\"3\",\"4\") = %v, want 7", v)
	}
}

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
