package bridge

import (
	"strings"
	"testing"
)

// TestMainPackageSource checks the vendoring rewrite: the bridge source must
// come out as `package main` with the cgo build constraint removed, so it can
// be dropped straight into a generated build directory.
func TestMainPackageSource(t *testing.T) {
	s := MainPackageSource()
	if strings.Contains(s, "//go:build cgo") {
		t.Error("vendored source still carries the //go:build cgo constraint")
	}
	if strings.Contains(s, "\npackage bridge\n") {
		t.Error("vendored source still declares package bridge")
	}
	if !strings.Contains(s, "\npackage main\n") {
		t.Error("vendored source does not declare package main")
	}
	// Sanity: the public entry points must survive the rewrite.
	for _, want := range []string{"func Init()", "func Import(", "func (o *Object) Call("} {
		if !strings.Contains(s, want) {
			t.Errorf("vendored source missing %q", want)
		}
	}
}
