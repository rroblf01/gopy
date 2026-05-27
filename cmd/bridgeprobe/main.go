//go:build cgo

// Command bridgeprobe exercises the embedded-CPython bridge end to end:
// it imports a real third-party library (pydantic_core), builds a schema
// validator, validates a Go map through it, and prints the round-tripped
// result. Success proves the hybrid architecture is viable; failure tells
// us to abandon the libpython-embed direction.
package main

import (
	"fmt"
	"os"

	"github.com/rroblf01/gopy/bridge"
)

func main() {
	if err := bridge.Init(); err != nil {
		die("init", err)
	}

	// --- 1. plain stdlib round-trip: import math, call sqrt(16) -----------
	math, err := bridge.Import("math")
	if err != nil {
		die("import math", err)
	}
	r, err := math.CallMethod("sqrt", 16.0)
	if err != nil {
		die("math.sqrt", err)
	}
	v, err := r.Go()
	if err != nil {
		die("sqrt->go", err)
	}
	fmt.Printf("math.sqrt(16) = %v (%T)\n", v, v)

	// --- 2. pydantic_core: build an int schema validator and validate -----
	pc, err := bridge.Import("pydantic_core")
	if err != nil {
		die("import pydantic_core", err)
	}
	ver, err := pc.Attr("__version__")
	if err == nil {
		if vs, e := ver.Go(); e == nil {
			fmt.Printf("pydantic_core __version__ = %v\n", vs)
		}
	}

	// SchemaValidator({"type": "int"}) then .validate_python("42")
	schemaValidator, err := pc.Attr("SchemaValidator")
	if err != nil {
		die("attr SchemaValidator", err)
	}
	schema := map[string]any{"type": "int"}
	validator, err := schemaValidator.Call(schema)
	if err != nil {
		die("SchemaValidator(schema)", err)
	}
	out, err := validator.CallMethod("validate_python", "42")
	if err != nil {
		die("validate_python('42')", err)
	}
	gout, err := out.Go()
	if err != nil {
		die("validate result->go", err)
	}
	fmt.Printf("pydantic int-validate('42') = %v (%T)\n", gout, gout)

	// --- 3. validation failure path: feed a non-int, expect a Python error -
	_, err = validator.CallMethod("validate_python", "not-a-number")
	if err != nil {
		fmt.Printf("expected validation error: %s\n", firstLine(err.Error()))
	} else {
		fmt.Println("UNEXPECTED: bad input validated cleanly")
	}

	fmt.Println("BRIDGE_OK")
}

func die(stage string, err error) {
	fmt.Fprintf(os.Stderr, "bridgeprobe: %s: %v\n", stage, err)
	os.Exit(1)
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
