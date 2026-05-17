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
