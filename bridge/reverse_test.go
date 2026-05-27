//go:build cgo

package bridge

import "testing"

// TestReverseBridge exposes a Go callback to Python and has Python invoke it
// indirectly via builtins.map — proving the Python→Go direction works.
func TestReverseBridge(t *testing.T) {
	doubler, err := RegisterFunc(func(args []any) any {
		return args[0].(int64) * 2
	})
	if err != nil {
		t.Fatalf("RegisterFunc: %v", err)
	}
	defer doubler.DecRef()

	builtins, err := Import("builtins")
	if err != nil {
		t.Fatalf("import builtins: %v", err)
	}
	defer builtins.DecRef()

	// list(map(doubler, [1, 2, 3])) — Python's map calls our Go func per item.
	mapped, err := builtins.CallMethod("map", doubler, []any{int64(1), int64(2), int64(3)})
	if err != nil {
		t.Fatalf("map(doubler, ...): %v", err)
	}
	defer mapped.DecRef()
	out, err := builtins.CallMethod("list", mapped)
	if err != nil {
		t.Fatalf("list(map(...)): %v", err)
	}
	defer out.DecRef()

	v, err := out.Go()
	if err != nil {
		t.Fatalf("result -> go: %v", err)
	}
	got, ok := v.([]any)
	if !ok || len(got) != 3 || got[0] != int64(2) || got[1] != int64(4) || got[2] != int64(6) {
		t.Fatalf("map(doubler, [1,2,3]) = %v, want [2 4 6]", v)
	}
}

// TestReverseBridgeDirect calls the Go-backed callable straight from Go via
// the forward Call path — the round trip Go→(Python callable wrapping Go)→Go.
func TestReverseBridgeDirect(t *testing.T) {
	greeter, err := RegisterFunc(func(args []any) any {
		return "hi " + args[0].(string)
	})
	if err != nil {
		t.Fatalf("RegisterFunc: %v", err)
	}
	defer greeter.DecRef()

	r, err := greeter.Call("ana")
	if err != nil {
		t.Fatalf("call greeter: %v", err)
	}
	defer r.DecRef()
	v, _ := r.Go()
	if v != "hi ana" {
		t.Fatalf("greeter('ana') = %v, want 'hi ana'", v)
	}
}

// TestReverseBridgePanic confirms a panicking Go callback surfaces as a
// Python error rather than crashing the interpreter.
func TestReverseBridgePanic(t *testing.T) {
	boom, err := RegisterFunc(func(args []any) any {
		panic("boom from go")
	})
	if err != nil {
		t.Fatalf("RegisterFunc: %v", err)
	}
	defer boom.DecRef()

	if _, err := boom.Call(); err == nil {
		t.Fatal("expected error from panicking callback, got nil")
	}
}
