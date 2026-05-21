# gopy

Python → Go transpiler written in Go. Reads a typed Python source file (or directory), emits idiomatic Go source, and lets you build a single statically-linked binary with `go build`. The end goal is shipping Python applications — eventually including Django apps — as compact native binaries with predictable RAM use.

## Status

Early but real. Current language support covers a typed subset of Python sufficient for self-contained programs and small multi-file projects. The transpiler is exercised against a golden-file test suite that compares stdout of the original Python source (run under CPython) with stdout of the transpiled Go binary.

The transpiler is intentionally **library-agnostic**: no code in `ir/`, `transpile/`, or the bundled stdlib shims is dedicated to any single framework. Support for third-party libraries comes from transpiling the library's own Python source alongside the application — which means the library must be written in the supported subset (no metaclasses, no `__getattr__`, no `setattr` magic).

### Supported

- Functions, parameters, return values
- Control flow: `if`/`elif`/`else`, `while`, `for ... in range(...)`, `for x in <iterable>`
- Primitives: `int`, `float`, `complex`, `str`, `bool`, `None`
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
- Stdlib shims: `sys.argv`, `sys.exit(n)`, `os.getenv(k)`, `time.time()`, `time.sleep(s)`, `json.dumps(x)`, `json.loads(s)`, `datetime.datetime.now()` returns a Datetime supporting `.year/.month/.day/.hour/.minute/.second/.isoformat()` and `+ timedelta` / `- timedelta` / `dt - dt` arithmetic, `datetime.timedelta(days)`, `pathlib.Path(...)` with `.exists()` / `.is_file()` / `.is_dir()` / `.read_text()` / `.write_text(s)`, `re.findall(p, s)`, `re.search(p, s)`, `re.match(p, s)`, `re.sub(p, repl, s)` — `search` and `match` return a Match object exposing `.group([n])` and `.groups()`; `x is None` / `x is not None` and truthy `if m:` checks work on these nullable returns; `csv.reader(lines)` parses a list of CSV lines into a `list[list[str]]`; `math` (`pi`, `e`, `inf`, `sqrt`, `floor`, `ceil`, `log`, `log2`, `log10`, `exp`, `sin`, `cos`, `tan`, `atan`, `atan2`, `pow`); `random.random()` / `random.randint(a, b)` / `random.seed(n)` (CPython and Go use different PRNGs, so deterministic-output fixtures must compare ranges, not values); `hashlib.sha256(b).hexdigest()` / `hashlib.sha1(b).hexdigest()` / `hashlib.sha512(b).hexdigest()` / `hashlib.md5(b).hexdigest()`; `base64.b64encode(b)` / `base64.b64decode(s)` (str in/out — `.encode()` / `.decode()` are no-ops); `from urllib.parse import quote, unquote`; `from collections import Counter` (typed dict-of-counts inline); `from itertools import chain, accumulate` (eagerly materialized as lists); `str.encode()` / `bytes.decode()` (no-op pass-through, since gopy uses `string` for both); `urllib.parse.urlencode(d)` for `dict[str, str]` (keys sorted for deterministic output); `string` constants (`ascii_lowercase`, `ascii_uppercase`, `ascii_letters`, `digits`, `hexdigits`, `octdigits`, `punctuation`, `whitespace`, `printable`); `collections.defaultdict(factory)` when the assignment carries a `dict[K, V]` annotation (the factory is ignored — Go's zero value covers missing keys); `subprocess.run(["cmd", ...])` returning a `CompletedProcess` with `returncode` / `stdout` / `stderr` attributes (CPython kwargs like `capture_output=True` / `text=True` are accepted at the call site and silently ignored)
- Context manager: `with open(path[, mode]) as fh:` — `fh.read()` and `fh.write(s)`
- Decorators: `@staticmethod` on free functions (no-op), `@property` on class methods (call sites emit `instance.attr` as a method invocation; properties are inherited via base lookup), `@classmethod` (emits a free `<Class>_<method>` function; calls of the form `Class.method(...)` dispatch to it, and `cls(...)` inside the body routes through the class's constructor)
- Default parameter values: `def f(a: int, b: int = 5)` — defaults are evaluated at every call site (so mutable defaults can't leak between calls)
- Keyword arguments at call sites for free functions **and instance methods**: `f(a, c=3, b=2)` or `obj.m(a, c=3, b=2)` reorders to match the parameter list and fills missing tail params from defaults
- `*args` and `**kwargs` capture in function signatures (typed as `[]any` and `map[string]any`); extra positionals/keywords at call sites flow into them
- List comprehensions `[expr for var in iter [if cond]]`, dict comprehensions `{k: v for var in iter [if cond]}`, and generator expressions `(expr for var in iter [if cond])` (multiple `for`+`if` clauses supported; the gen-expr form materializes eagerly)
- `min` / `max` with multiple positional args (`min(a, b, c)`) on int/float/str
- `print(..., sep=..., end=...)` kwargs override the default space separator and trailing newline
- Forward-reference annotations: `-> "MyClass"` resolves to the named type
- Union annotations (`int | str`, `typing.Union[...]`, `typing.Optional[T]`) lower to `any`; combine with `isinstance` to narrow at the call site. **isinstance narrowing** is honored: `if isinstance(x, Foo): x.attr` shadows `x` inside the body with the narrowed type, so attribute / method access dispatches correctly (works for user classes, `int`, `float`, `str`, `bool`)
- Extended typing aliases accepted: `Final[T]` / `ClassVar[T]` / `Annotated[T, ...]` unwrap to their inner type; `Mapping[K, V]` / `MutableMapping[K, V]` lower to `dict[K, V]`; `MutableSequence[T]` / `Collection[T]` / `Iterator[T]` / `Reversible[T]` lower to `list[T]`; `Type[T]` / `Self` / `Final` / `ClassVar` / `Never` / `NoReturn` / `TypeAlias` (bare) lower to `any`
- Annotated attribute stores in method bodies: `self.field: list[int] = []` parses and emits as a regular attribute assignment
- Class-level field annotations on regular (non-dataclass) classes — `class Bag:\n    items: list[int]\n    label: str\n    def __init__(self, label: str): ...` declares typed struct fields without needing `@dataclass`; `ClassVar[T]` annotations are accepted and dropped from the struct. Empty literal initializers (`self.items = []`) cast automatically to the declared field type
- Class-level field **defaults**: `class Config:\n    name: str = "default"\n    timeout: int = 30` apply at the head of `__init__`, before any user-written body. If `__init__` overrides the field, the user's assignment wins. Classes with defaults and no explicit `__init__` get a synthesized one that applies the defaults
- `complex(re, im)` builtin backed by Go's native `complex128` — `c.real` / `c.imag` map to `real(c)` / `imag(c)`; arithmetic operators (`+ - * /`) work natively. `print(c)` formats as Python's `(re+imj)` (or `imj` for pure-imaginary) rather than Go's `(re+imi)`. `abs(c)` returns magnitude (float64)
- `cmath` module: `sqrt`, `exp`, `log`, `log10`, `sin`/`cos`/`tan`, `asin`/`acos`/`atan`, `sinh`/`cosh`/`tanh`, `phase(c)`, `polar(c)` → `[r, phi]`, `rect(r, phi)`, `isnan`, `isinf`, `isfinite`, constants `pi`/`e`/`tau`/`inf`/`nan`/`infj`/`nanj` — backed by Go's `math/cmplx`
- Bare annotation statements (`x: int` with no initializer) accepted: declares the type in scope without emitting a statement, so the name is typed for downstream usage without tripping Go's "declared but not used" rule
- Lambdas as first-class values with a `Callable[[A, B, ...], R]` annotation: `f: Callable[[int], int] = lambda x: x * 2`. The annotation drives a re-lower of the lambda body with concrete param types and the assignment emits a typed Go function value (`var f func(int64) int64 = ...`), so the body can use type-specific ops. Without the annotation, lambdas still fall back to `func(any) any` which doesn't compile for non-trivial bodies
- Generic functions (PEP 695 syntax): `def first[T](xs: list[T]) -> T: return xs[0]` lowers to `func first[T any](xs []T) T { return xs[0] }`. Multiple type params (`def pair[A, B](a: A, b: B)`) are supported. Only free functions are generic — Go methods can't introduce new type parameters separately from their receiver
- Standalone `enumerate(xs[, start])` and `zip(xs, ys)` — return `[][]any` (list of 2-elt pair slices). Tuple unpacking inside a `for` still uses the optimized Go-native range path; the standalone form materializes pairs eagerly so they can be passed around as values
- `match` class patterns (keyword form): `case Point(x=0, y=0): ...` / `case Circle(r=1): ...` / `case Point():` — type-asserts the subject against the class pointer, then checks each named field against the literal pattern. Positional captures (`Point(x, y)`) aren't supported yet; rewrite as keyword form. Literal value, singleton, wildcard, and `|` patterns still work as before
- `match` sequence patterns (fixed length): `case []:` / `case [0]:` / `case [1, 2, 3]:` — emits `len(__subj) == N && __subj[0] == v0 && ...`. Star unpacking (`case [first, *rest]`) and nested patterns aren't supported yet
- `match` mapping patterns (partial-match): `case {"x": 0}:` / `case {"x": 1, "y": 2}:` — each `(key, value)` pair must be present in the subject dict with the matching value; extra keys are ignored, matching CPython's semantics. `**rest` capture isn't supported. `case {}:` matches any mapping (acts as a default)
- No-op decorators accepted on free functions and methods: `@final`, `@override`, `@no_type_check`, `@deprecated` (and `typing.` / `warnings.` qualified forms), plus `@typing.overload` / `@overload` (the stub gets dropped entirely so the real impl wins). Class bodies tolerate `__slots__ = (...)`, `__match_args__ = (...)`, `__all__ = [...]`, and bare `_ = ...` statements without rejecting the class
- Reflective class attributes: `obj.__class__` returns the gopy `__Type` wrapper (same shape as `type(obj)`); `obj.__class__.__name__`, `Foo.__name__`, and `type(obj).__name__` all yield the class name as a string
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
- HTTP GET: `urllib.request.urlopen(url)` returns an `HTTPResponse`-like wrapper with `.read()` (body as str), `.status`, `.headers` (dict), `.close()`, `.getcode()` — minimal subset, no POST yet
- `html.escape(s)` (replaces `& < > " '`) and `html.unescape(s)` (via Go's `html.UnescapeString` for the full entity set)
- `secrets.token_hex(n)` / `secrets.token_urlsafe(n)` / `secrets.token_bytes(n)` / `secrets.randbelow(n)` — backed by `crypto/rand`
- `platform.system()` / `platform.machine()` / `platform.node()` / `platform.release()` / `platform.python_version()` / `platform.platform()` — values mapped from Go's `runtime.GOOS` / `runtime.GOARCH` (e.g. `darwin` → `Darwin`, `amd64` → `x86_64`); `python_version()` returns a fixed stub ("3.12.0") since there's no embedded CPython
- `socket.gethostbyname(host)` — backed by Go's `net.LookupHost`; raises `gaierror`-tagged Exception on failure
- `os.environ` — exposed as a `dict[str, str]` snapshot of the process environment (annotate the binding `env: dict[str, str] = os.environ` to keep `in` / `.get()` working). Read-only — mutations don't propagate back to the OS. Also: `os.cpu_count()`, `os.urandom(n)`, `os.path.isabs(p)`, `os.path.lexists(p)` (includes broken symlinks)
- `sys.platform` (`runtime.GOOS`), `sys.byteorder` (`"little"`), `sys.maxsize`, `sys.version`, `sys.version_info` (5-tuple stub `(3, 12, 0, "final", 0)`)
- `copy.copy(v)` / `copy.deepcopy(v)` — round-trip through `encoding/json`. Works for primitives, lists/dicts of JSON-friendly values; deeper graphs and class instances need manual cloning. The return type erases to `any`, so a typed annotation on the target is required to keep using slice/dict ops on the result
- `queue.Queue()` / `queue.LifoQueue()` — single-typed `*__Queue` wrapper around a slice + `sync.Mutex`. Methods: `.put(v)`, `.get()` (raises `queue.Empty`-tagged Exception on empty), `.qsize()`, `.empty()`, `.full()` (always False — no maxsize wired yet). Goroutine-safe but `get()` doesn't block

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

_Generated by `scripts/update_bench.sh` on 2026-05-21T10:52:52Z._

| Benchmark | CPython (ms) | gopy Go (ms) | Speedup | Python RSS (MB) | gopy RSS (MB) | RSS save |
|-----------|-------------:|-------------:|--------:|----------------:|--------------:|---------:|
| bench_class | 47.13 | 1.62 | 29.1x | 12.86 | 5.06 | 2.54x |
| bench_fib | 134.10 | 5.37 | 25.0x | 12.83 | 4.88 | 2.63x |
| bench_loop | 106.63 | 1.71 | 62.4x | 12.96 | 5.03 | 2.58x |

_Hardware: Linux 6.18.32-1-lts x86_64. Go: go1.26.3-X:nodwarf5. Python: 3.14.5._

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
- [x] Augmented assignment for non-numeric types: `s += "x"`, `lst += [x]`, `d |= other` (Python 3.9+ dict merge)
- [x] Pure `d1 | d2` dict union expression (right-side wins on key collisions)
- [x] `s % args` printf-style formatting (single arg or tuple of args; uses Go's `fmt.Sprintf`)
- [x] `ascii(x)` builtin (string repr with non-ASCII escaped as `\xHH` / `\uHHHH` / `\UHHHHHHHH`)
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
- [x] Module-level `name = expr` / `name: T = expr` declarations (emit as Go package-scope vars)
- [x] `global` declaration inside functions (writes route to the package var, no shadow); `nonlocal` accepted (best-effort scope binding)
- [ ] Class methods that return new instances of the class without forward-reference annotations
- [ ] Allow `return` inside `try` blocks (currently scoped to the IIFE wrapper)

### Dynamic features

- [x] `getattr(obj, name[, default])` and `setattr(obj, name, value)` via per-class accessor helpers (no reflection needed; receiver type must be statically known)
- [x] `hasattr(obj, name)`
- [ ] `isinstance(obj, Cls)` and `issubclass(...)` with the generated class hierarchy
- [x] `type(obj)` returning a `<class 'name'>`-stringifying handle with `.__name__` (no `is` / `==` comparison against class literals — use `isinstance` for that)
- [ ] Metaclasses (`class Foo(metaclass=Meta):`) — limited to compile-time hooks; full runtime metaclasses are out of scope
- [ ] `__getattr__` / `__setattr__` / `__getattribute__`
- [ ] Descriptors (`__get__` / `__set__`)
- [ ] Dynamic attribute creation (`self.__dict__[name] = value`)

### Type system

- [ ] Full type inference pass (forward + backward propagation) so plain `x = expr` rarely needs an annotation
- [x] Union types (`int | str`) lowered to `any` with `isinstance` dispatch
- [x] `typing.Optional[T]` / `typing.Union[...]` / `typing.List[T]` / `typing.Dict[K, V]` / `typing.Tuple[...]` / `typing.Set[T]` / `typing.Iterable[T]` / `typing.Sequence[T]` / bare `typing.Any` / `typing.Callable` (collapse to `any`); `bytes` annotation lowers to `str`
- [ ] Generic functions and classes (`def first(xs: list[T]) -> T:`)
- [ ] Protocols / structural typing via Go interfaces
- [ ] Narrowing through `isinstance` / `is None` checks
- [x] True division (`int / int → float`), floor division (`int // int → int`), and int↔float promotion on `+`/`-`/`*`/`/`

### Standard library

- [x] `re.Pattern` objects (`p = re.compile(...); p.match(s)` / `.search` / `.fullmatch` / `.findall` / `.sub` / `.subn` / `.split`)
- [x] `re.search` / `re.match` / `re.compile(...).search` named groups (`m.group("name")`, `m.groupdict()`); `m.start([g])`, `m.end([g])`, `m.span([g])` positions
- [x] `re.split(pattern, s)`, `re.escape(s)`, `re.fullmatch(pattern, s)`, `re.subn(pattern, repl, s)` returns `[result, count]`
- [x] `csv.writer(fh)` stateful writer with `.writerow(row)` / `.writerows(rows)` bound to a `with open(...)` handle; `csv.DictReader(lines)` returns `[]map[string]string`; `csv.DictWriter(fh, fields)` with `.writeheader()` / `.writerow(d)` / `.writerows(rows)`
- [x] `pathlib.Path` arithmetic (`p / "sub"`), `.name`, `.parent`, `.suffix`, `.stem`
- [x] `pathlib.Path.iterdir()` (returns `[]*Path`; loop var inherits the Path tag so `for child in p.iterdir(): print(child.name)` works), `.mkdir(parents, exist_ok)`, `.unlink()`, `.glob(pattern)` (shell-style glob via `filepath.Glob`)
- [x] `json.dumps(v, indent=N)` pretty-prints with N-space indentation; default form keeps the existing Python-style separators
- [x] `json.load(fh)` / `json.dump(v, fh)` for `with open(...) as fh:` handles
- [x] File iteration line by line (`for line in fh:`)
- [x] `datetime.timedelta` keyword constructor with full parameter set (`days`, `seconds`, `microseconds`, `milliseconds`, `minutes`, `hours`, `weeks`)
- [x] `datetime.datetime.strptime(s, fmt)` and `.strftime(fmt)` / `date.strftime(fmt)` (Python format codes `%Y/%m/%d/%H/%M/%S/%y/%B/%b/%A/%a/%p/%j/%z` mapped to Go's reference-time layout)
- [x] `datetime.datetime.fromtimestamp(ts)`, `.fromisoformat(s)`, `.utcnow()`, `.weekday()`, `.isoweekday()`, `.timestamp()`, `.replace(year=..., ...)`, `datetime.combine(date, time)`; `date.replace(year=..., month=..., day=...)`
- [x] `timedelta.total_seconds()`, `.days`, `.seconds`, `abs(td)`, `td + td`, `td - td`, `td * int`, `int * td`, `td / int`
- [x] `enum.auto()` (sequential integer assignment, mixes with explicit values)
- [x] `@dataclass` field defaults via `field(default=...)` / `field(default_factory=list/dict/set)` — fresh container per instance
- [x] `dataclasses.asdict(obj)` (returns `map[string]any`), `dataclasses.astuple(obj)` (returns `[]any`), `dataclasses.replace(obj, **kwargs)` (fresh instance via constructor), `dataclasses.fields(cls_or_obj)` (returns `[]string` of field names)
- [x] `dict.items()` standalone (returns `[]struct{Key, Value}`); for-loop tuple-unpack form unchanged
- [x] `sum(xs, start)` 2-arg form (int/float; promotes to float when either side is float)
- [x] `datetime.date(y, m, d)` / `datetime.time(h, m, s)` with `.year/.month/.day` (`.hour/.minute/.second`), `.isoformat()`, `.weekday()`, `.isoweekday()`; `date.today()`, `date.fromisoformat(s)`
- [x] `subprocess.run` returning a typed `CompletedProcess` (kwargs ignored)
- [ ] `argparse` (or a lightweight subset) for CLI tooling
- [x] `logging` levels writing to stderr (no level filtering, no handlers); `logging.getLogger(name)` returns a `__Logger` with `.debug`/`.info`/`.warning`/`.error`/`.critical` prefixed by the logger name
- [x] `collections.Counter`
- [x] `collections.defaultdict` (annotation-driven; factory ignored)
- [x] `collections.deque` (list-backed; append/appendleft/pop/popleft)
- [x] `collections.OrderedDict` (annotation-driven; treated as a plain `dict[K, V]` since Python 3.7+ already preserves insertion order)
- [x] `collections.namedtuple("Name", [...])` at module level — synthesizes a struct with `any`-typed fields and a `Name(args)` constructor; accepts both list-of-names and space-separated string forms
- [x] `itertools.chain`, `itertools.chain.from_iterable`, `itertools.accumulate`, `itertools.takewhile`, `itertools.dropwhile`, `itertools.combinations` (r=2), `itertools.permutations` (r=2), `itertools.product` (2-way), `itertools.islice(it, [start,] stop[, step])`, `itertools.repeat(value, n)` (bounded form), `itertools.starmap(lambda, pairs)`, `itertools.filterfalse(lambda, xs)`, `itertools.compress(data, selectors)`, `itertools.count(start, step, n)` (bounded form), `itertools.zip_longest(a, b[, fillvalue=...])`
- [x] `itertools.groupby` (with optional `key=` lambda; consecutive grouping like CPython)
- [ ] full-arity `combinations` / `product`
- [x] `functools.reduce` (binary lambda + optional initializer)
- [x] `functools.lru_cache`, `functools.cache`, `functools.cached_property`, `functools.wraps`, `functools.singledispatch` (accepted as no-op decorators)
- [x] `functools.partial` (free functions only; produces a typed Go closure)
- [x] `math` (trig, asin/acos/atan/atan2, sinh/cosh/tanh + asinh/acosh/atanh, log/log2/log10/log1p, exp/expm1, sqrt, floor/ceil-as-int, trunc, fmod, gcd, lcm, isnan/isinf/isfinite/isclose, copysign, hypot, degrees/radians, factorial, comb, perm, dist, prod (int64), remainder, erf/erfc/gamma/lgamma, constants `pi`/`e`/`tau`/`inf`/`nan`)
- [x] `random` (`random()`, `randint(a, b)`, `uniform(a, b)`, `seed(n)`, `choice(xs)`, `shuffle(xs)` in-place, `sample(xs, k)`)
- [x] `statistics` (`mean`, `fmean`, `median`, `median_low`, `median_high`, `mode`, `variance`, `pvariance`, `stdev`, `pstdev`, `harmonic_mean`)
- [x] `heapq` (`heappush`, `heappop`, `heapify`, `heappushpop`, `nsmallest`, `nlargest`) — min-heap on typed int/float/str lists
- [x] `bisect` (`bisect_left`, `bisect_right`, `bisect`, `insort`, `insort_left`, `insort_right`) — typed binary search / insertion
- [x] `uuid.uuid4()` returns a hyphenated lowercase hex string (RFC 4122 v4 layout)
- [x] `textwrap.dedent`, `textwrap.indent(s, prefix)`, `textwrap.fill(s, width)` (width-only form)
- [x] `secrets.token_hex([n])`, `secrets.token_urlsafe([n])`, `secrets.token_bytes([n])` (CSPRNG via `crypto/rand`)
- [x] `getpass.getuser()` (looks up `LOGNAME` / `USER` / `USERNAME` env vars)
- [x] `threading.Lock()` / `threading.RLock()` (single-goroutine no-op shims with `.acquire()` / `.release()` / `.locked()`)
- [x] `typing.cast(T, x)` — runtime identity (returns `x` unchanged; type annotation is informational only)
- [x] `time.monotonic()`, `time.perf_counter()`, `time.time_ns()` (monotonic nanosecond reading)
- [x] `shutil.rmtree(p)`, `shutil.copy(src, dst)` / `shutil.copyfile`, `shutil.move(src, dst)`
- [x] `tempfile.mkdtemp([prefix])`, `tempfile.mkstemp([prefix])` (returns `[fd, name]`), `tempfile.gettempdir()`
- [x] `hmac.new(key, msg, digestmod)` (sha1/sha256/sha512/md5), `.update(msg)`, `.hexdigest()`, `hmac.compare_digest`
- [x] `gzip.compress(s)` / `gzip.decompress(s)`, `zlib.compress(s)` / `zlib.decompress(s)` / `zlib.crc32(s)` / `zlib.adler32(s)`
- [x] `glob.glob(pattern)` (annotate the receiving var as `list[str]` so methods like `.sort()` resolve)
- [x] `socket.gethostname()` / `socket.getfqdn()`
- [x] `b"..."` bytes literals pass through as `str` (gopy uses Go's `string` for both)
- [x] `string` constants (`ascii_letters`, `digits`, `punctuation`, ...)
- [x] `s.format(...)` positional `{}` placeholders + format specs (`{:5d}`, `{:05d}`, `{:.2f}`, `{:>10}`, `{:<10}`, `{:^10}`, `{:*>6}`, `{:x}`, `{:08x}`, `{:b}`)
- [x] f-string format specs (`f"{x:.2f}"`, `f"{n:05d}"`, etc.) and `!r` / `!s` / `!a` conversions
- [x] `format(value[, spec])` builtin (single-value formatter; same spec mini-language as `str.format`)
- [x] Bitwise operators on ints: `|`, `&`, `^`, `<<`, `>>`, `~` (unary)
- [x] `hashlib.sha256` / `hashlib.md5`
- [x] `base64.b64encode` / `b64decode`, `urlsafe_b64encode` / `urlsafe_b64decode`, `b32encode` / `b32decode`, `b16encode` / `b16decode`
- [x] `urllib.parse.quote` / `unquote` / `quote_plus` / `unquote_plus` / `urlencode` / `urlparse` / `parse_qs` / `parse_qsl`
- [ ] `http.server` and `http.client` (or `urllib.request`)

### Builtins

- [x] `map(fn, xs)` and `filter(fn, xs)` — when `fn` is an inline lambda
- [x] `zip(a, b)` and `enumerate(xs[, start])` in `for` loops
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
- [x] `hex(n)` / `oct(n)` / `bin(n)` with Python's `0x` / `0o` / `0b` prefix and `-` for negative inputs
- [x] `callable(x)` (static for known function/class names, reflect-based fallback for runtime values)
- [x] `vars(obj)` (field map; same shape as `dataclasses.asdict`), `dir(obj_or_cls)` (typed `[]string` of field + method names)
- [x] `eval` / `exec` / `compile` rejected at transpile time with a clear error (no runtime interpreter)
- [x] `string.Template("$name")` with `.substitute(d)` (KeyError on missing) / `.safe_substitute(d)`, `string.capwords(s[, sep])`
- [x] `set(iter)` / `frozenset(iter)` — return a deduplicated typed slice (insertion-order preserved; not a true hash-set)
- [x] `list.count(x)`, `list.index(x)` (raises ValueError when missing), `list.extend(ys)`, `list.insert(i, v)`, `list.remove(v)`, `list.clear()`, `list.sort([reverse=True, key=lambda])`, `list.reverse()`, `list.copy()`, `list.pop([i])` (IndexError on empty / out-of-range; negative indices supported)
- [x] `str.isdigit()`, `.isalpha()`, `.isalnum()`, `.isspace()`, `.isupper()`, `.islower()`, `.isnumeric()`, `.isdecimal()`, `.isidentifier()`, `.isprintable()`, `.isascii()`
- [x] `str.split(sep, maxsplit)`, `str.splitlines()`, `str.partition(sep)`, `str.rpartition(sep)`, `str.removeprefix(p)`, `str.removesuffix(s)`
- [x] `str.startswith` / `str.endswith` accept a tuple of candidates (short-circuit chained check)
- [x] `str.maketrans(from, to[, delete])` + `str.translate(table)` — rune→string mapping
- [x] `str.casefold()`, `str.swapcase()`, `str.expandtabs([tabsize])`
- [x] `min(xs, key=lambda x: ...)` / `max(xs, key=lambda x: ...)` — re-lower lambda with element type, pick element with min/max key
- [x] `"x" * n` / `n * "x"` string repetition (routes to `strings.Repeat`)
- [x] `[v] * n` / `n * [v]` list repetition (typed IIFE that appends the slice n times)
- [x] `round(x, ndigits)` 2-arg form (returns float, scales by 10^ndigits)
- [x] `os.getcwd()`, `os.listdir(p)`, `os.makedirs(p[, exist_ok])`, `os.mkdir(p)`, `os.rmdir(p)`, `os.remove(p)`, `os.rename(src, dst)`, `os.sep`, `os.linesep`
- [x] `os.path.join`, `os.path.exists`, `os.path.isfile`, `os.path.isdir`, `os.path.basename`, `os.path.dirname`, `os.path.splitext`, `os.path.abspath`, `os.path.split`, `os.path.relpath`, `os.path.getsize`, `os.path.normpath`, `os.path.expanduser`, `os.path.expandvars`, `os.path.commonprefix`, `os.path.samefile`
- [x] `calendar.isleap(y)`, `calendar.monthrange(y, m)` (returns `[weekday, ndays]`), `calendar.weekday(y, m, d)`, `calendar.month_name[i]` / `month_abbr[i]` / `day_name[i]` / `day_abbr[i]` (index into typed slices)
- [x] `dict.update(other)`, `dict.pop(key[, default])` (raises KeyError when missing and no default), `dict.setdefault(key, default)`, `dict.clear()`, `dict.copy()`, `dict.popitem()` (returns `[]any{key, value}`), `dict.fromkeys(iter[, value])` (target annotation needed: `d: dict[K, V] = dict.fromkeys(...)`)

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
- [x] Dunder methods on user classes (`__str__` → Go `String()`, `__repr__` → `Repr()`, `__len__` → `Len()`, `__hash__` → `Hash()`); explicit `obj.__str__()` calls rewrite to the Go names, so `print(obj)` triggers the Stringer dispatch automatically
- [x] Operator overloading via dunder methods: `__add__` / `__sub__` / `__mul__` / `__truediv__` / `__floordiv__` / `__mod__` / `__pow__` / `__or__` / `__and__` / `__xor__` / `__lshift__` / `__rshift__` / `__matmul__` and `__lt__` / `__le__` / `__gt__` / `__ge__` / `__eq__` / `__ne__` route `a + b` / `a < b` / `a | b` etc. through the method when LHS is a registered user class; unary `-obj` / `+obj` / `~obj` route through `__neg__` / `__pos__` / `__invert__`; augmented assignment (`obj += x`, `obj |= x`, …) tries the in-place dunder (`__iadd__`, `__ior__`, …) first and falls back to the regular dunder
- [x] User-class context managers: `with Resource(...) as r: body` routes through `__enter__` / `__exit__`; teardown runs via `defer` so it fires even on panic. The `as` binding lives only inside the `with` block (Go closure scope), which differs from CPython where it persists in the enclosing scope
- [x] `assert cond[, msg]` — falsy condition panics with an `AssertionError` exception carrying the formatted message; truthiness uses Python semantics (empty containers/strings, zero, `None` are all falsy). Always-on (no `-O` mode switch)
- [x] `abc.ABC` / `abc.ABCMeta` base classes and `@abstractmethod` / `@abc.abstractmethod` / `@abstractclassmethod` / `@abstractstaticmethod` / `@abstractproperty` — accepted. **Pure-abstract** classes (ABC base + only `@abstractmethod` methods + no fields + no `__init__`) emit as a Go `interface`, so subclasses satisfy the type structurally and a parameter annotated `s: Shape` accepts any subclass instance. Heterogeneous `list[Shape]` literals (`[Square(2.0), Circle(1.0)]`) cast each element into the interface at literal-emit time. Mixed classes (concrete + abstract methods) still emit as a struct, with abstract stubs panicking with `NotImplementedError`
- [x] `typing.Protocol` / `typing.Generic` / `typing.Generic[T]` / `typing.Protocol[T]` accepted as marker bases (no Go embed). Bare-name forms drop from the base list — gopy uses Go's structural typing instead of runtime Protocol checks
- [x] Bare expression statements that are just `None` / `...` / docstrings drop from the lowered body so `def f(): ...` abstract stubs and module docstrings compile cleanly
- [x] Container dunder methods: `__getitem__` (drives `obj[k]`), `__setitem__` (drives `obj[k] = v`), `__contains__` (drives `x in obj` / `x not in obj`)
- [x] Builtin dispatch to dunders: `bool(obj)` → `__bool__`, `len(obj)` → `__len__`, `abs(obj)` → `__abs__`, `str(obj)` → `__str__`, `int(obj)` → `__int__`, `float(obj)` → `__float__`, `reversed(obj)` → `__reversed__`, `hash(obj)` → `__hash__`, `round(obj[, n])` → `__round__`, `math.ceil(obj)` / `math.floor(obj)` / `math.trunc(obj)` → `__ceil__` / `__floor__` / `__trunc__`, `obj(args)` → `__call__`, and `for v in obj:` → `__iter__` when the method returns a typed list (also recognized: `__next__`)
- [x] f-string `__format__` dispatch: `f"{obj:spec}"` calls `obj.__format__("spec")` when the class defines it (empty spec also routes through the dunder, matching CPython)

### Hard / open questions

- [ ] Runtime model that supports both static Go performance and Python-style dynamic typing where unavoidable (`any` fallback with type-switched dispatch)
- [ ] Memory model: when can we use values vs. pointers? When can we stack-allocate?
- [ ] Concurrency model: should generators become bounded channels by default? How do we surface goroutine leaks?
- [ ] Garbage collection: how to convey that long-lived Python globals become package-level Go vars without leaking goroutines from generators
- [ ] Multi-file project shape: per-package vs. flat-namespace tradeoffs

## License

MIT — see [LICENSE](LICENSE).
