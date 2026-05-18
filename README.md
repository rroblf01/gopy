# gopy

Python → Go transpiler written in Go. Reads a typed Python source file (or directory), emits idiomatic Go source, and lets you build a single statically-linked binary with `go build`. The end goal is shipping Python applications — eventually including Django apps — as compact native binaries with predictable RAM use.

## Status

Early but real. Current language support covers a typed subset of Python sufficient for self-contained programs and small multi-file projects. The transpiler is exercised against a golden-file test suite that compares stdout of the original Python source (run under CPython) with stdout of the transpiled Go binary.

The transpiler is intentionally **library-agnostic**: no code in `ir/`, `transpile/`, or the bundled stdlib shims is dedicated to any single framework. Support for third-party libraries comes from transpiling the library's own Python source alongside the application — which means the library must be written in the supported subset (no metaclasses, no `__getattr__`, no `setattr` magic).

### Supported

- Functions, parameters, return values
- Control flow: `if`/`elif`/`else`, `while`, `for ... in range(...)`, `for x in <iterable>`
- Primitives: `int`, `float`, `str`, `bool`, `None`
- Collections: `list[T]`, `dict[K, V]` — literals, indexing, `len(...)`, `.append(...)`, augmented assignment, `dict.get(key, default)`
- Classes with `__init__`, instance methods, attributes
- Single **and multiple** inheritance, `super().__init__(...)`, `super().method(...)`; mixin bases are zero-initialized automatically. Diamond conflicts (same method inherited from two bases without override) are caught at transpile time with a clear error.
- Exceptions: `try` / `except E as e` / `finally`, `raise E("msg")`, custom exception classes inheriting `Exception`
- f-strings: `f"x = {x}"` → `fmt.Sprintf`
- Builtins: `print`, `len`, `str`, `int`, `float`, `bool`, `range`, `sorted`, `sum`, `min`, `max`, `any`, `all`, `reversed`, `abs`, `round`, `isinstance` (single class or tuple-of-classes), `issubclass`, `pow`, `chr`, `ord`, `repr`, `divmod`, `getattr`, `setattr`, `hasattr`, `list` (iterator materialization — pass-through in the gopy shim), `iter`, `next(it[, default])`, `id`, `hash`
- Slicing: `xs[a:b]`, `xs[a:]`, `xs[:b]`, `xs[:]`, `xs[a:b:step]`, and negative bounds (`xs[-3:]`, `xs[::-1]`) — routed through a runtime helper for non-trivial cases, fast path for simple bounds
- Tuple literals as values (`pair = (1, 2)`) — lowered to a slice; iteration and indexing work
- Set literals `{1, 2, 3}` — lower to the same slice shape; `in` / `not in` work, uniqueness not enforced
- `in` / `not in` operators on strings (`strings.Contains`), dicts (comma-ok lookup), and lists (inline scan with element-type cast)
- Augmented list concat (`xs += ys`) → `append(xs, ys...)`
- String methods: `.upper()`, `.lower()`, `.strip([chars])`, `.lstrip([chars])`, `.rstrip([chars])`, `.split([sep])`, `sep.join(parts)`, `.replace(old, new)`, `.startswith(s)`, `.endswith(s)`, `.find(s)`, `.count(sub)`, `.title()`, `.capitalize()`, `.center(width[, fillchar])`, `.ljust(width[, fillchar])`, `.rjust(width[, fillchar])`, `.zfill(width)` — chained calls infer through return types
- Type inference of user-function and user-method return types: `b = make_box(7)` propagates the declared `Box` return type onto `b` so `b.method()` dispatches correctly without an annotation
- `break` and `continue` inside loops
- Ternary expression: `x if cond else y`
- Chained comparisons: `lo <= n <= hi` desugars to `(lo <= n) and (n <= hi)`
- Tuple unpacking on assignment (`a, b = 1, 2`, `a, b = b, a`) and in two-name `for` loops (`for i, x in enumerate(xs):`, `for k, v in d.items():`, `for x, y in zip(xs, ys):`)
- Chained assignment: `a = b = c = 0` lowers to a sequence of single assigns
- File iteration: `for line in fh:` over a handle opened with `with open(...)` — backed by `bufio.Scanner`, trailing newline stripped
- `str.format("{} ...")` with positional `{}` placeholders
- `dict.keys()` / `dict.values()` standalone return slices; `.items()` only via for-loop unpacking
- Multi-file projects in a single directory (each `.py` → sibling `.go`, shared `package main`)
- Stdlib shims: `sys.argv`, `sys.exit(n)`, `os.getenv(k)`, `time.time()`, `time.sleep(s)`, `json.dumps(x)`, `json.loads(s)`, `datetime.datetime.now()` returns a Datetime supporting `.year/.month/.day/.hour/.minute/.second/.isoformat()` and `+ timedelta` / `- timedelta` / `dt - dt` arithmetic, `datetime.timedelta(days)`, `pathlib.Path(...)` with `.exists()` / `.is_file()` / `.is_dir()` / `.read_text()` / `.write_text(s)`, `re.findall(p, s)`, `re.search(p, s)`, `re.match(p, s)`, `re.sub(p, repl, s)` — `search` and `match` return a Match object exposing `.group([n])` and `.groups()`; `x is None` / `x is not None` and truthy `if m:` checks work on these nullable returns; `csv.reader(lines)` parses a list of CSV lines into a `list[list[str]]`; `math` (`pi`, `e`, `inf`, `sqrt`, `floor`, `ceil`, `log`, `log2`, `log10`, `exp`, `sin`, `cos`, `tan`, `atan`, `atan2`, `pow`); `random.random()` / `random.randint(a, b)` / `random.seed(n)` (CPython and Go use different PRNGs, so deterministic-output fixtures must compare ranges, not values); `hashlib.sha256(b).hexdigest()` and `hashlib.md5(b).hexdigest()`; `base64.b64encode(b)` / `base64.b64decode(s)` (str in/out — `.encode()` / `.decode()` are no-ops); `from urllib.parse import quote, unquote`; `from collections import Counter` (typed dict-of-counts inline); `from itertools import chain, accumulate` (eagerly materialized as lists); `str.encode()` / `bytes.decode()` (no-op pass-through, since gopy uses `string` for both); `urllib.parse.urlencode(d)` for `dict[str, str]` (keys sorted for deterministic output); `string` constants (`ascii_lowercase`, `ascii_uppercase`, `ascii_letters`, `digits`, `hexdigits`, `octdigits`, `punctuation`, `whitespace`); `collections.defaultdict(factory)` when the assignment carries a `dict[K, V]` annotation (the factory is ignored — Go's zero value covers missing keys); `subprocess.run(["cmd", ...])` returning a `CompletedProcess` with `returncode` / `stdout` / `stderr` attributes (CPython kwargs like `capture_output=True` / `text=True` are accepted at the call site and silently ignored)
- Context manager: `with open(path[, mode]) as fh:` — `fh.read()` and `fh.write(s)`
- Decorators: `@staticmethod` on free functions (no-op), `@property` on class methods (call sites emit `instance.attr` as a method invocation; properties are inherited via base lookup), `@classmethod` (emits a free `<Class>_<method>` function; calls of the form `Class.method(...)` dispatch to it, and `cls(...)` inside the body routes through the class's constructor)
- Default parameter values: `def f(a: int, b: int = 5)` — defaults are evaluated at every call site (so mutable defaults can't leak between calls)
- Keyword arguments at call sites for free functions **and instance methods**: `f(a, c=3, b=2)` or `obj.m(a, c=3, b=2)` reorders to match the parameter list and fills missing tail params from defaults
- `*args` and `**kwargs` capture in function signatures (typed as `[]any` and `map[string]any`); extra positionals/keywords at call sites flow into them
- List comprehensions `[expr for var in iter [if cond]]`, dict comprehensions `{k: v for var in iter [if cond]}`, and generator expressions `(expr for var in iter [if cond])` (single generator, optional filter; the gen-expr form materializes eagerly)
- `min` / `max` with multiple positional args (`min(a, b, c)`) on int/float/str
- `print(..., sep=..., end=...)` kwargs override the default space separator and trailing newline
- Forward-reference annotations: `-> "MyClass"` resolves to the named type
- Union annotations (`int | str`, `typing.Union[...]`, `typing.Optional[T]`) lower to `any`; combine with `isinstance` to narrow at the call site
- `typing.List[T]` / `typing.Dict[K, V]` aliases (same lowering as the lowercase forms)
- Walrus assignment in `if` / `while` conditions: `if (n := f()) > 0:` hoists the binding into a preceding `Assign` so `n` survives into the body; the while form re-evaluates each iteration
- Multi-return functions: declaring `-> tuple[T, U]` emits a Go multi-value return, and `a, b = f()` consumes it directly without an intermediate slice
- `@lru_cache` / `@lru_cache(maxsize=...)` / `@functools.lru_cache(...)` decorators are accepted and treated as no-ops (the wrapped function still runs uncached)
- Generators: functions with `yield` compile to a Go goroutine + receive-only channel; `for x in gen():` reads from the channel; `yield from inner_gen()` forwards each value to the outer channel
- `collections.deque([items])` constructs a `*__Deque` (list-backed) with `.append` / `.appendleft` / `.pop` / `.popleft` methods that panic with `IndexError` on empty
- Lambda expressions, with specialization at well-known call sites: `map(lambda x: ..., xs)`, `filter(lambda x: ..., xs)`, `sorted(xs, key=lambda x: ..., reverse=...)`, and `functools.reduce(lambda a, b: ..., xs[, init])` re-lower the body with the iterable's element type so arithmetic / attribute access typechecks. Standalone lambdas fall back to `func(p any) any` and only work when the body operates on `any`-friendly values.
- `logging.basicConfig(**kwargs)` (kwargs ignored), `logging.debug` / `info` / `warning` / `error` / `critical` — each writes a `LEVEL:root:msg` line to stderr to match CPython's default formatter
- `from <stdlib> import <name>` aliases (e.g. `from datetime import datetime`)
- Auto-detection of project venv (`./.venv/bin/python3`, `./venv/bin/python3`) when running the transpiler — override with `-python` or the `GOPY_PYTHON` env var
- Pythonic `print()` rendering of `True`/`False`/`None`
- Dynamic attribute access via `getattr` / `setattr` / `hasattr` on user-class instances. Every transpiled class gets generated `__<Class>_get` / `__<Class>_set` helpers that switch over the declared fields — no Go reflection required. The receiver's class must be statically known (annotation, constructor return, etc.).
- `match` / `case` statement: literal patterns, `|` alternation, `_` wildcard, and `if`-guarded `_` arms. Lowered to a chained `if/else if` over a single evaluation of the subject. Sequence / mapping / class / `as`-capture patterns are not yet supported.
- `itertools.takewhile(predicate, xs)` and `dropwhile(predicate, xs)` — predicate must be an inline lambda; the element type is inferred from the iterable
- `itertools.combinations(xs, 2)` and `itertools.product(a, b)` — only the fixed two-element / two-iterable forms; both materialize as `list[list[T]]`
- Nested function definitions (closures): `def inner(...):` inside another function lowers to `name := func(...) ret { ... }`, capturing the enclosing scope via Go closure semantics
- `f(*xs)` splat at call sites when the target function declares `*args` — the typed slice is wrapped into a `[]any` inline
- URL parsing: `urllib.parse.urlparse(s)` returns a `ParseResult` with `.scheme` / `.netloc` / `.path` / `.params` / `.query` / `.fragment`

### Not yet supported

- Custom decorators (user-written `@my_wrapper`) and decorators with arguments — only built-in `@staticmethod` / `@classmethod` / `@property` are accepted
- `csv.writer` as a stateful file-bound writer; today only `csv.reader(lines)` round-trips
- `timedelta(seconds=..., minutes=...)` keyword constructors (only positional days)
- Metaclasses, `__getattr__` / descriptors
- Dynamic features: `eval`, `exec`, monkey-patching, runtime `setattr`
- Generator `send()` / `throw()`, async / `await` — these require a fundamentally different runtime model (coroutine state machine vs. plain goroutines) and are intentionally out of scope for v1
- C extensions
- Library frameworks that depend on dynamic Python features (Django, Flask, SQLAlchemy, ...). The transpiler stays library-agnostic: third-party library code must itself be written in the supported Python subset so it can be transpiled alongside the application code.

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
- `-python <path>` — Python interpreter to invoke (default: `./.venv/bin/python3` / `./venv/bin/python3` if present, else `python3` on `PATH`). Honors `GOPY_PYTHON`.

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

_Generated by `scripts/update_bench.sh` on 2026-05-18T12:05:41Z._

| Benchmark | CPython (ms) | gopy Go (ms) | Speedup | Python RSS (MB) | gopy RSS (MB) | RSS save |
|-----------|-------------:|-------------:|--------:|----------------:|--------------:|---------:|
| bench_class | 46.56 | 1.56 | 29.8x | 12.85 | 4.60 | 2.79x |
| bench_fib | 136.00 | 5.41 | 25.1x | 12.93 | 4.56 | 2.84x |
| bench_loop | 106.49 | 1.81 | 58.8x | 12.82 | 4.54 | 2.82x |

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

## Roadmap

High-level checklist of what still needs to land before gopy is genuinely usable for production-style Python projects. Items are roughly grouped by area; within each group they're ordered by how foundational they are.

### Language semantics

- [x] Slicing with explicit step and negative bounds
- [x] Tuple unpacking on assignment + in `for` loops via `enumerate` / `zip` / `dict.items()`
- [x] Tuple literal as a value (lowered to a slice — immutability not enforced)
- [x] Multiple-target chained assignment (`a = b = 0`)
- [x] `break` and `continue` statements
- [x] Conditional expression (ternary): `x if cond else y`
- [x] Chained comparisons (`a < b < c`)
- [ ] Augmented assignment for non-numeric types (`s += "x"` already works; `lst += [x]` and `d |= other` do not)
- [x] `f(*xs)` splat for varargs targets
- [x] `f(**d)` splat for kwargs targets (mixes with explicit kwargs)
- [x] `yield from` (forwards each value of an inner iterable through the outer channel)
- [x] Generator expressions `(expr for x in xs)` (materialized eagerly)
- [x] Walrus operator (`:=`) inside `if` / `while` conditions
- [x] `match`/`case` statement (literal + `|` + `_` + guards; structural patterns deferred)
- [ ] `async def` / `await` / `async for` / `async with`
- [ ] Custom decorators (user-written `@wrap`) and decorators with arguments (`@route("/path")`)
- [ ] Decorators on class methods other than the built-in three
- [x] Inner / nested function definitions with closure capture
- [x] Lambda expressions (specialized inside `map` / `filter` / `sorted(key=...)`)
- [ ] Lambdas as first-class values (closures with full type inference)
- [ ] `global` / `nonlocal` declarations
- [ ] Class methods that return new instances of the class without forward-reference annotations
- [ ] Allow `return` inside `try` blocks (currently scoped to the IIFE wrapper)

### Dynamic features

- [x] `getattr(obj, name[, default])` and `setattr(obj, name, value)` via per-class accessor helpers (no reflection needed; receiver type must be statically known)
- [x] `hasattr(obj, name)`
- [ ] `isinstance(obj, Cls)` and `issubclass(...)` with the generated class hierarchy
- [ ] `type(obj)` returning a comparable class handle
- [ ] Metaclasses (`class Foo(metaclass=Meta):`) — limited to compile-time hooks; full runtime metaclasses are out of scope
- [ ] `__getattr__` / `__setattr__` / `__getattribute__`
- [ ] Descriptors (`__get__` / `__set__`)
- [ ] Dynamic attribute creation (`self.__dict__[name] = value`)

### Type system

- [ ] Full type inference pass (forward + backward propagation) so plain `x = expr` rarely needs an annotation
- [x] Union types (`int | str`) lowered to `any` with `isinstance` dispatch
- [x] `typing.Optional[T]` / `typing.Union[...]` / `typing.List[T]` / `typing.Dict[K, V]`
- [ ] Generic functions and classes (`def first(xs: list[T]) -> T:`)
- [ ] Protocols / structural typing via Go interfaces
- [ ] Narrowing through `isinstance` / `is None` checks
- [ ] Better numeric promotion (`int / int → float`, `int // int → int`, mixed `int + float`)

### Standard library

- [ ] `re.Pattern` objects (`p = re.compile(...); p.match(s)`)
- [ ] `re.search` named groups (`m.group("name")`)
- [ ] `csv.writer` as a stateful file-bound writer with `.writerow()` / `.writerows()`
- [ ] `pathlib.Path` arithmetic (`p / "sub"`), iteration (`p.iterdir()`), and globbing
- [x] File iteration line by line (`for line in fh:`)
- [ ] `datetime.timedelta` keyword constructor and arithmetic with all units
- [ ] `datetime.date` / `datetime.time` standalone types
- [x] `subprocess.run` returning a typed `CompletedProcess` (kwargs ignored)
- [ ] `argparse` (or a lightweight subset) for CLI tooling
- [x] `logging` levels writing to stderr (no level filtering, no handlers)
- [x] `collections.Counter`
- [x] `collections.defaultdict` (annotation-driven; factory ignored)
- [x] `collections.deque` (list-backed; append/appendleft/pop/popleft)
- [ ] `collections.OrderedDict`
- [x] `itertools.chain`, `itertools.accumulate`, `itertools.takewhile`, `itertools.dropwhile`, `itertools.combinations` (r=2), `itertools.product` (2-way)
- [x] `itertools.groupby` (with optional `key=` lambda; consecutive grouping like CPython)
- [ ] full-arity `combinations` / `product`
- [x] `functools.reduce` (binary lambda + optional initializer)
- [x] `functools.lru_cache` (accepted as no-op decorator)
- [x] `functools.partial` (free functions only; produces a typed Go closure)
- [x] `math` (trig, log, sqrt, floor/ceil-as-int, constants)
- [x] `random` (`random()`, `randint(a, b)`, `seed(n)`)
- [x] `string` constants (`ascii_letters`, `digits`, `punctuation`, ...)
- [x] `s.format(...)` positional `{}` placeholders
- [x] `hashlib.sha256` / `hashlib.md5`
- [x] `base64.b64encode` / `base64.b64decode`
- [x] `urllib.parse.quote` / `unquote` / `urlencode` / `urlparse`
- [ ] `http.server` and `http.client` (or `urllib.request`)

### Builtins

- [x] `map(fn, xs)` and `filter(fn, xs)` — when `fn` is an inline lambda
- [x] `zip(a, b)` and `enumerate(xs)` in `for` loops
- [ ] `zip` / `enumerate` returning standalone iterables
- [x] `reversed(xs)`
- [x] `abs`, `round`, `pow`
- [x] `divmod`
- [x] `chr`, `ord`
- [x] `repr` (approximate — uses Go's `%#v` formatter)
- [x] `isinstance` (single class or tuple of classes)
- [x] `issubclass` (resolved at transpile time from the recorded base chain)
- [x] `id(obj)` returning a stable integer (via FNV hash of the value's `%v` form)
- [x] `hash(obj)`
- [x] `iter(iter)` (pass-through), `next(it[, default])` over generator channels
- [x] `set(iter)` / `frozenset(iter)` — return a deduplicated typed slice (insertion-order preserved; not a true hash-set)

### Tooling and infrastructure

- [ ] Single-shot CLI that takes a `.py` and writes a built binary (`gopy build script.py -o script`)
- [ ] Project mode: detect `pyproject.toml` / `requirements.txt` and produce a Go module + go.sum
- [ ] Watch mode (`gopy watch src/`) that rebuilds on change
- [ ] Source maps / line directives so panic stacks reference the Python source
- [ ] LSP / editor diagnostics: report unsupported features at edit time
- [ ] Stricter transpile errors with caret pointers at the unsupported construct
- [ ] CI workflow in this repo (GitHub Actions) running the fixture suite
- [ ] Continuous benchmarks dashboard so regressions surface in PRs

### Codegen quality

- [ ] Pluggable target packages: emit into multiple Go files / subpackages to mirror Python module layout
- [ ] Avoid emitting unused `_ = args` / `_ = kwargs` stubs when the variable is actually used
- [ ] Reuse helper functions across the program (today each one is emitted at the bottom of the single output file)
- [ ] Optional `unsafe` / inlined fast paths for hot loops
- [ ] Generate `String()` / `Format()` methods on user classes so `print(obj)` matches `repr(obj)` more closely

### Hard / open questions

- [ ] Runtime model that supports both static Go performance and Python-style dynamic typing where unavoidable (`any` fallback with type-switched dispatch)
- [ ] Memory model: when can we use values vs. pointers? When can we stack-allocate?
- [ ] Concurrency model: should generators become bounded channels by default? How do we surface goroutine leaks?
- [ ] Garbage collection: how to convey that long-lived Python globals become package-level Go vars without leaking goroutines from generators
- [ ] Multi-file project shape: per-package vs. flat-namespace tradeoffs

## License

MIT — see [LICENSE](LICENSE).
