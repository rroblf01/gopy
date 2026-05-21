#!/usr/bin/env python3
"""Dump Python source AST as JSON for gopy transpiler."""
import ast
import json
import sys


def node_to_dict(node):
    if isinstance(node, ast.AST):
        result = {"_type": type(node).__name__}
        for field, value in ast.iter_fields(node):
            result[field] = node_to_dict(value)
        if hasattr(node, "lineno"):
            result["lineno"] = node.lineno
        if hasattr(node, "col_offset"):
            result["col_offset"] = node.col_offset
        # Annotate Constant nodes with their Python primitive type so the
        # Go decoder can distinguish `1` (int) from `1.0` (float) — both
        # land as JSON numbers otherwise.
        if isinstance(node, ast.Constant):
            v = node.value
            if isinstance(v, bool):
                result["_const_kind"] = "bool"
            elif isinstance(v, int):
                result["_const_kind"] = "int"
            elif isinstance(v, float):
                result["_const_kind"] = "float"
            elif isinstance(v, str):
                result["_const_kind"] = "str"
            elif isinstance(v, bytes):
                # Pass bytes literals through as str — gopy uses Go's
                # string for both. Replace the value field so the lowerer
                # sees a plain string.
                result["_const_kind"] = "str"
                result["value"] = v.decode("utf-8", "replace")
            elif v is None:
                result["_const_kind"] = "none"
            elif v is Ellipsis:
                # `...` placeholder body for abstract methods / stubs.
                # Lower as None so it disappears from generated code.
                result["_const_kind"] = "none"
                result["value"] = None
            elif isinstance(v, complex):
                # Imaginary / complex literals (`2j`, `1+2j`). Encode as
                # a (real, imag) pair the Go side reassembles into a
                # ComplexLit node.
                result["_const_kind"] = "complex"
                result["value"] = {"real": v.real, "imag": v.imag}
        return result
    if isinstance(node, list):
        return [node_to_dict(x) for x in node]
    if isinstance(node, (str, int, float, bool)) or node is None:
        return node
    if isinstance(node, bytes):
        return {"_type": "Bytes", "value": node.decode("utf-8", "replace")}
    return {"_type": "Unknown", "repr": repr(node)}


def main():
    if len(sys.argv) < 2:
        src = sys.stdin.read()
        filename = "<stdin>"
    else:
        filename = sys.argv[1]
        with open(filename, "r", encoding="utf-8") as f:
            src = f.read()
    tree = ast.parse(src, filename=filename)
    json.dump(node_to_dict(tree), sys.stdout)


if __name__ == "__main__":
    main()
