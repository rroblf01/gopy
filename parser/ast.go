// Package parser loads a Python AST as raw JSON (produced by scripts/py_ast_dump.py)
// and exposes it as a generic Node tree. Higher layers (ir) lower this into typed IR.
package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Node is a single Python AST node. Fields mirror the Python ast module:
// "_type" holds the node class name; other keys are the node's fields.
type Node map[string]any

func (n Node) Type() string {
	if t, ok := n["_type"].(string); ok {
		return t
	}
	return ""
}

func (n Node) Lineno() int {
	if v, ok := n["lineno"].(float64); ok {
		return int(v)
	}
	return 0
}

// Child returns the named field as a Node, or nil if absent/not a node.
func (n Node) Child(name string) Node {
	v, ok := n[name]
	if !ok || v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return Node(m)
	}
	return nil
}

// Children returns the named field as a []Node when the field is a list of nodes.
func (n Node) Children(name string) []Node {
	v, ok := n[name]
	if !ok || v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]Node, 0, len(arr))
	for _, x := range arr {
		if m, ok := x.(map[string]any); ok {
			out = append(out, Node(m))
		}
	}
	return out
}

// Str returns the named field as a string (or "" if missing/wrong type).
func (n Node) Str(name string) string {
	if v, ok := n[name].(string); ok {
		return v
	}
	return ""
}

// ParseFile runs the Python dumper on path and returns the Module node.
func ParseFile(dumperPath, srcPath string) (Node, error) {
	cmd := exec.Command("python3", dumperPath, srcPath)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("python AST dump failed: %v: %s", err, errBuf.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("decode AST json: %w", err)
	}
	return Node(raw), nil
}
