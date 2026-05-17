// Package runtime is the Go-side support library imported by transpiled
// programs. Generated code references this package for shared types
// (exceptions, builtin shims) so multiple compiled modules agree on layout.
package runtime

import "fmt"

// Exception is the base type behind Python's `Exception` hierarchy.
// User-defined Python exceptions inheriting Exception embed *Exception
// in their generated struct, so a single `recover()` value can be
// pattern-matched against any subclass with a type assertion.
type Exception struct {
	Msg string
}

func NewException(msg string) *Exception { return &Exception{Msg: msg} }

func (e *Exception) Error() string { return e.Msg }

// String makes Exception print nicely via fmt.Println.
func (e *Exception) String() string {
	if e == nil {
		return "<nil>"
	}
	return e.Msg
}

// Format implements fmt.Formatter so `print(e)` and f-strings reproduce
// CPython's behavior of printing just the message.
func (e *Exception) Format(s fmt.State, verb rune) {
	if e == nil {
		fmt.Fprint(s, "<nil>")
		return
	}
	fmt.Fprint(s, e.Msg)
}
