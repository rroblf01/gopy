# gopy

Python → Go transpiler written in Go. Reads a typed Python source file (or directory), emits idiomatic Go source, and lets you build a single statically-linked binary with `go build`. The end goal is shipping Python applications — eventually including Django apps — as compact native binaries with predictable RAM use.

## Status

Early but real. Current language support covers a typed subset of Python sufficient for self-contained programs and small multi-file projects. The transpiler is exercised against a golden-file test suite that compares stdout of the original Python source (run under CPython) with stdout of the transpiled Go binary.

### Supported (phases F1–F4)

- Functions, parameters, return values
- Control flow: `if`/`elif`/`else`, `while`, `for ... in range(...)`, `for x in <iterable>`
- Primitives: `int`, `float`, `str`, `bool`, `None`
- Collections: `list[T]`, `dict[K, V]` — literals, indexing, `len(...)`, `.append(...)`, augmented assignment
- Classes with `__init__`, instance methods, attributes
- Single inheritance, `super().__init__(...)`, `super().method(...)`
- Exceptions: `try` / `except E as e` / `finally`, `raise E("msg")`, custom exception classes inheriting `Exception`
- f-strings: `f"x = {x}"` → `fmt.Sprintf`
- Builtins: `print`, `len`, `str`, `int`, `float`, `range`
- Multi-file projects in a single directory (each `.py` → sibling `.go`, shared `package main`)
- Stdlib shims (F4): `sys.argv`, `sys.exit(n)`, `os.getenv(k)`, `time.time()`, `time.sleep(s)`
- Context manager (F4): `with open(path[, mode]) as fh:` — `fh.read()` and `fh.write(s)`
- Decorators (F4): `@staticmethod` on free functions (no-op, accepted for compatibility)
- Generators (F4): functions with `yield` compile to a Go goroutine + receive-only channel; `for x in gen():` reads from the channel
- Pythonic `print()` rendering of `True`/`False`/`None`

### Not yet supported

- Richer stdlib (`json`, `re`, `datetime`, file I/O beyond read/write)
- Method decorators (`@staticmethod` / `@classmethod` / `@property` inside a class)
- Multiple inheritance, metaclasses, `__getattr__` / descriptors
- Dynamic features: `eval`, `exec`, monkey-patching, runtime `setattr`
- Generator `send()` / `throw()`, async / `await`
- C extensions
- Django itself (the long-term target, gated on richer stdlib + ORM + templating shims)

## Requirements for the input Python code

For a Python file to transpile cleanly, it must obey the following rules:

1. **Type hints are mandatory** on every function parameter, return value, and class attribute initialization. The transpiler does not infer signatures; missing annotations become `any` and almost always trigger a Go build error downstream.
2. **Use only the supported subset** listed above. Anything else raises a `gopy: line N: unsupported ...` error at transpile time.
3. **Entry point**: wrap top-level execution in `if __name__ == "__main__":`. The transpiler ignores that block (Go calls `main()` automatically); without it, CPython would not invoke `main()` and tests would mismatch.
4. **Exceptions** must derive from `Exception` and accept a `msg: str` in `__init__` if you want `str(e)` to round-trip across Python and Go.
5. **Multi-file projects**: place every `.py` in a single directory. `from sibling import name` is dropped at lowering time — names resolve via Go's shared package namespace.
6. **Stdlib imports**: only the modules listed under *Supported* are recognized. `import sys` / `import os` / `import time` are accepted; anything else falls through and produces an undefined-name error.
7. **No `return` inside a `try` block** — the IIFE wrapper would only return from the wrapper, not the enclosing function.
8. **`with open(...)` per block**: variables declared inside a `with` block are scoped to that block (IIFE). If you need a value after the block, do the work inside the same block, or read into a variable declared before the `with`.

## Installation

```bash
git clone https://github.com/rroblf01/gopy.git
cd gopy
go build ./...
```

You need:

- Go ≥ 1.22
- Python 3.10+ (used as an out-of-process AST dumper; no runtime dependency in the generated binary)

## Usage

### Transpile a single file

```bash
go run ./cmd/gopy -o out.go tests/fixtures/fib.py
go build -o fib out.go
./fib
```

Flags:

- `-o <file>` — write Go source to file (default: stdout)
- `-pkg <name>` — Go package name (default: `main`)
- `-dumper <path>` — explicit path to `scripts/py_ast_dump.py` (auto-located by default)

### Transpile a whole directory

```bash
go run ./cmd/gopy-build -o out_dir tests/fixtures_multi/calc
cd out_dir
go mod init myapp && go build -o app .
./app
```

Every `.py` in the source directory becomes a sibling `.go` file in the output directory, all sharing the chosen Go package.

### Run the benchmark harness

```bash
go run ./cmd/gopy-bench -n 5 tests/fixtures/bench_fib.py
```

It transpiles, compiles, then runs both the CPython script and the resulting Go binary `n` times, reporting min/median/mean wall time and peak RSS.

### Run the test suite

```bash
go test ./tests/...
```

Each fixture under `tests/fixtures/` is executed via CPython and via the transpiled binary; stdout must match exactly.

### Refresh the benchmark table

```bash
scripts/update_bench.sh
```

Re-runs every `bench_*.py` fixture and rewrites the **Benchmarks** section below in place.

## Benchmarks

CPython 3.x vs. the `gopy`-transpiled Go binary, on identical CPU-bound workloads. Each row reports the mean of 5 timed runs after 1 warmup run. Lower wall time = faster; lower RSS = less RAM.

<!-- BENCH_START -->

_Generated by `scripts/update_bench.sh` on 2026-05-17T09:53:14Z._

| Benchmark | CPython (ms) | gopy Go (ms) | Speedup | Python RSS (MB) | gopy RSS (MB) | RSS save |
|-----------|-------------:|-------------:|--------:|----------------:|--------------:|---------:|
| bench_class | 48.05 | 1.69 | 28.4x | 12.91 | 4.03 | 3.20x |
| bench_fib | 134.76 | 5.35 | 25.2x | 12.90 | 4.09 | 3.15x |
| bench_loop | 107.27 | 1.78 | 60.3x | 12.92 | 4.19 | 3.08x |

_Hardware: Linux 6.18.31-1-lts x86_64. Go: go1.26.3-X:nodwarf5. Python: 3.14.5._

<!-- BENCH_END -->

## Repository layout

```
gopy/
├── cmd/
│   ├── gopy/         # single-file transpiler CLI
│   ├── gopy-build/   # directory transpiler CLI
│   └── gopy-bench/   # benchmark runner
├── parser/           # Python AST loader (subprocess + JSON)
├── ir/               # typed IR + AST→IR lowering
├── transpile/        # IR → Go source
├── runtime/          # Go-side support library (Exception base, etc.)
├── scripts/
│   ├── py_ast_dump.py    # Python AST dumper invoked by parser
│   └── update_bench.sh   # refreshes the README benchmark table
└── tests/
    ├── fixtures/         # single-file golden tests
    ├── fixtures_multi/   # multi-file project tests
    └── integration_test.go
```

## License

MIT — see [LICENSE](LICENSE).
