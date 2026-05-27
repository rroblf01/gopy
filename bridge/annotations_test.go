//go:build cgo

package bridge

import "testing"

// TestRichAnnotations checks the eval-based annotation resolver handles the
// generic / Optional forms real handlers use, and that a Go-registered type
// name resolves in a signature.
func TestRichAnnotations(t *testing.T) {
	// def f(xs: list[int], name: Optional[str]) -> dict[str, int]
	f, err := RegisterTypedFunc("f",
		[]Param{
			{Name: "xs", Annotation: "list[int]"},
			{Name: "name", Annotation: "Optional[str]"},
		},
		"dict[str, int]",
		func(args []any, kwargs map[string]any) any { return map[string]any{"n": int64(1)} },
	)
	if err != nil {
		t.Fatalf("RegisterTypedFunc: %v", err)
	}
	defer f.DecRef()

	inspectMod, err := Import("inspect")
	if err != nil {
		t.Fatalf("import inspect: %v", err)
	}
	defer inspectMod.DecRef()
	sig, err := inspectMod.CallMethod("signature", f)
	if err != nil {
		t.Fatalf("signature(f): %v", err)
	}
	defer sig.DecRef()
	s, _ := sig.Str()
	// Python 3.14 renders list[int] as "list[int]" and Optional[str] as
	// "str | None"; assert the structural pieces are present.
	for _, want := range []string{"xs: list[int]", "str | None", "-> dict[str, int]"} {
		if !containsSub(s, want) {
			t.Fatalf("signature %q missing %q", s, want)
		}
	}
}

// TestRegisterType registers a Go-declared class under a name and uses it as
// an annotation, confirming the resolver picks it up.
func TestRegisterType(t *testing.T) {
	// A trivial class declared from Go.
	cls, err := MakeClass("Widget", nil, []Field{{Name: "id", Annotation: "int"}})
	if err != nil {
		t.Fatalf("MakeClass(Widget): %v", err)
	}
	defer cls.DecRef()
	if err := RegisterType("Widget", cls); err != nil {
		t.Fatalf("RegisterType: %v", err)
	}

	// def handle(w: Widget) -> bool
	h, err := RegisterTypedFunc("handle",
		[]Param{{Name: "w", Annotation: "Widget"}}, "bool",
		func(args []any, kwargs map[string]any) any { return true },
	)
	if err != nil {
		t.Fatalf("RegisterTypedFunc(handle): %v", err)
	}
	defer h.DecRef()

	typingMod, err := Import("typing")
	if err != nil {
		t.Fatalf("import typing: %v", err)
	}
	defer typingMod.DecRef()
	hints, err := typingMod.CallMethod("get_type_hints", h)
	if err != nil {
		t.Fatalf("get_type_hints(handle): %v", err)
	}
	defer hints.DecRef()
	// The 'w' hint should be the Widget class — check its __name__.
	wHint, err := hints.GetItem("w")
	if err != nil {
		t.Fatalf("hints['w']: %v", err)
	}
	defer wHint.DecRef()
	nameObj, err := wHint.Attr("__name__")
	if err != nil {
		t.Fatalf("Widget.__name__: %v", err)
	}
	defer nameObj.DecRef()
	nv, _ := nameObj.Go()
	if nv != "Widget" {
		t.Fatalf("handle's 'w' annotation = %v, want Widget", nv)
	}
}

func containsSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
