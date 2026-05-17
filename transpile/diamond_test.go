package transpile

import (
	"strings"
	"testing"

	"github.com/rroblf01/gopy/ir"
)

// TestDiamondConflictDetected ensures that a class inheriting the same
// method name from two distinct bases without overriding it fails fast
// with a clear error pointing at the Python-level names.
func TestDiamondConflictDetected(t *testing.T) {
	m := &ir.Module{
		Decls: []ir.Decl{
			&ir.Class{Name: "A", MethodNames: []string{"foo"}},
			&ir.Class{Name: "B", MethodNames: []string{"foo"}},
			&ir.Class{Name: "C", Bases: []string{"A", "B"}}, // does NOT override foo
		},
	}
	_, err := Module(m, Options{PackageName: "main"})
	if err == nil {
		t.Fatal("expected diamond conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "foo") || !strings.Contains(err.Error(), "C") {
		t.Fatalf("error %q should mention method 'foo' and class 'C'", err)
	}
}

// TestDiamondConflictResolvedByOverride accepts the same shape as long
// as the subclass explicitly defines the disputed method.
func TestDiamondConflictResolvedByOverride(t *testing.T) {
	m := &ir.Module{
		Decls: []ir.Decl{
			&ir.Class{Name: "A", MethodNames: []string{"foo"}},
			&ir.Class{Name: "B", MethodNames: []string{"foo"}},
			&ir.Class{Name: "C", Bases: []string{"A", "B"}, MethodNames: []string{"foo"}},
		},
	}
	if _, err := Module(m, Options{PackageName: "main"}); err != nil {
		t.Fatalf("override should suppress the conflict, got %v", err)
	}
}
