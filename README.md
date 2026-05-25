# gopy

Python → Go transpiler written in Go. Reads a typed Python source file (or directory), emits idiomatic Go source, and lets you build a single statically-linked binary with `go build`. The end goal is shipping Python applications — eventually including Django apps — as compact native binaries with predictable RAM use.

## Status

Early but real — a typed subset of Python sufficient for self-contained programs and small multi-file projects, exercised against a golden-file test suite that compares stdout of the original Python source (run under CPython) with stdout of the transpiled Go binary.

The transpiler is intentionally **library-agnostic**: no code in `ir/`, `transpile/`, or the bundled stdlib shims is dedicated to any single framework. Third-party libraries are supported by transpiling their own Python source alongside the application — so the library must itself be written in the supported subset (no metaclasses, no `__getattr__`, no `setattr` magic).

**Detailed feature inventory** (what's supported / not yet supported) lives in [STATUS.md](STATUS.md) — moved out of the README to keep this file scannable.

## Requirements for the input Python code

For a Python file to transpile cleanly, it must obey the following rules:

1. **Type hints are mandatory** on every function parameter, return value, and class attribute initialization. The transpiler does not infer signatures; missing annotations become `any` and almost always trigger a Go build error downstream.
2. **Use only the supported subset** listed above. Anything else raises a `gopy: line N: unsupported ...` error at transpile time.
3. **Entry point**: wrap top-level execution in `if __name__ == "__main__":`. The transpiler ignores that block (Go calls `main()` automatically); without it, CPython would not invoke `main()` and tests would mismatch.
4. **Exceptions** must derive from `Exception` and accept a `msg: str` in `__init__` if you want `str(e)` to round-trip across Python and Go.
5. **Multi-file projects**: place every `.py` in a single directory. `from sibling import name` is dropped at lowering time — names resolve via Go's shared package namespace.
6. **Stdlib imports**: only the modules listed under *Supported* are recognized. `import sys` / `import os` / `import time` are accepted; anything else falls through and produces an undefined-name error.
7. ~~No `return` inside a `try` block~~ — supported as of the try-return trap. `return` inside `try` / `except` / `finally` ferries the value out of the IIFE wrapper through generated `__try_retval_N` / `__try_ret_N` locals and re-returns from the enclosing function after the deferred recover/finally unwind.
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

### Build a single binary in one step

```bash
go run ./cmd/gopy build -o fib tests/fixtures/fib.py
./fib
```

The `build` subcommand transpiles the input into a temp directory, drops a minimal `go.mod`, and runs `go build` to produce a native executable. Pass `-keep` to retain the intermediate Go source dir for inspection.

### Watch and rebuild on change

```bash
go run ./cmd/gopy watch -o fib -interval 500ms tests/fixtures/fib.py
```

The `watch` subcommand polls the input mtime (no fsnotify dep) and re-runs `gopy build` whenever it changes. Build failures print to stderr and the loop keeps running so a transient bad save doesn't kill the watcher.

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

_Generated by `scripts/update_bench.sh` on 2026-05-22T17:19:49Z._

| Benchmark | CPython (ms) | gopy Go (ms) | Speedup | Python RSS (MB) | gopy RSS (MB) | RSS save |
|-----------|-------------:|-------------:|--------:|----------------:|--------------:|---------:|
| bench_class | 54.01 | 2.66 | 20.3x | 12.86 | 6.23 | 2.06x |
| bench_fib | 136.25 | 5.68 | 24.0x | 12.85 | 5.93 | 2.17x |
| bench_loop | 105.25 | 2.05 | 51.3x | 12.74 | 5.96 | 2.14x |

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
- [x] `async def` / `await` / `async for` / `async with` — gopy strips the async layer (sync semantics). `async for` lowers to the same range-over-iterable as `for`; `async with` accepts `__aenter__` / `__aexit__` as aliases of the sync pair. Real concurrency still requires hand-written goroutines
- [x] Custom name-form decorators (user-written `@wrap`) on free functions and methods — accepted as identity passthroughs. The decorator body is not executed, so semantics that depend on it (caching, logging, retry) won't be reproduced. Decorators with arguments (`@cache(maxsize=128)` / `@route("/path")`) and attribute-form decorators (`@mod.attr`) outside the recognized list are still rejected
- [x] Decorators on class methods (any name) — same passthrough treatment as free-function decorators
- [x] Inner / nested function definitions with closure capture
- [x] Lambda expressions (specialized inside `map` / `filter` / `sorted(key=...)`)
- [x] Lambdas as first-class values: lambdas passed to user functions with a `Callable[[T], U]` typed parameter re-lower their body against the declared param types at the call site, so `apply(lambda x: x * 2, 7)` compiles even when `apply` is user-defined. The standalone-lambda fallback (`f = lambda x: ...` without annotation) still emits `func(p any) any` and only works for `any`-friendly bodies
- [x] Module-level `name = expr` / `name: T = expr` declarations (emit as Go package-scope vars)
- [x] `global` declaration inside functions (writes route to the package var, no shadow); `nonlocal` accepted (best-effort scope binding)
- [x] Class methods that return new instances of the class — both the forward-reference form (`-> "Box"`) and the bare-name form under `from __future__ import annotations` (`-> Box`) resolve to the enclosing class so `b.doubled().n` typechecks downstream
- [x] `return` inside `try` / `except` / `finally` blocks — when any return is present, the IIFE wrapper stages the value into local `__try_retval_N` / `__try_ret_N` and the enclosing function re-returns after the deferred finally / recover unwind

### Dynamic features

- [x] `getattr(obj, name[, default])` and `setattr(obj, name, value)` via per-class accessor helpers (no reflection needed; receiver type must be statically known)
- [x] `hasattr(obj, name)`
- [x] `isinstance(obj, Cls)` (single class or tuple of classes) and `issubclass(Sub, Base)` — both resolve against the generated class hierarchy at transpile time. Inheritance walks the lowered base chain so subclass-check semantics match CPython
- [x] `type(obj)` returning a `<class 'name'>`-stringifying handle with `.__name__` (no `is` / `==` comparison against class literals — use `isinstance` for that)
- [ ] Metaclasses (`class Foo(metaclass=Meta):`) — limited to compile-time hooks; full runtime metaclasses are out of scope
- [x] `__getattr__` fallback on user classes: when `getattr(obj, name)` or the per-class accessor finds no declared field, gopy delegates to `obj.Getattr(name)`. `__setattr__` is also recognized (renamed to `Setattr`) and runs on writes to undeclared fields. Writes to *declared* fields still go straight to the struct slot rather than routing through `__setattr__` — that bit of CPython's semantics is not reproduced. `__getattribute__` (called on *every* access, including declared fields) remains out of scope
- [ ] Descriptors (`__get__` / `__set__`)
- [ ] Dynamic attribute creation (`self.__dict__[name] = value`)

### Type system

- [ ] Full type inference pass (forward + backward propagation) so plain `x = expr` rarely needs an annotation
- [x] Union types (`int | str`) lowered to `any` with `isinstance` dispatch
- [x] `typing.Optional[T]` / `typing.Union[...]` / `typing.List[T]` / `typing.Dict[K, V]` / `typing.Tuple[...]` / `typing.Set[T]` / `typing.Iterable[T]` / `typing.Sequence[T]` / bare `typing.Any` / `typing.Callable` (collapse to `any`); `bytes` annotation lowers to `str`
- [x] Generic functions (`def first[T](xs: list[T]) -> T`) — PEP 695 and classic `T = TypeVar("T")` forms both lower to Go generics. Generic classes (`class Box[T]:`) are not supported; use `Generic[T]` base or `any`-typed fields
- [x] Protocols / structural typing — `typing.Protocol` is a marker base. Pure-abstract `abc.ABC` classes (interface-only, no fields, no `__init__`) emit as Go `interface`s; subclasses satisfy them structurally
- [x] Narrowing through `isinstance` (single class, primitive, tuple) and `is None` / `is not None` — narrowed names shadow inside the branch so attribute / method access dispatches against the asserted type
- [x] True division (`int / int → float`), floor division (`int // int → int`), and int↔float promotion on `+`/`-`/`*`/`/`

### Standard library

- [x] `re.Pattern` objects (`p = re.compile(...); p.match(s)` / `.search` / `.fullmatch` / `.findall` / `.sub` / `.subn` / `.split`)
- [x] `re.search` / `re.match` / `re.compile(...).search` named groups (`m.group("name")`, `m.groupdict()`); `m.start([g])`, `m.end([g])`, `m.span([g])` positions
- [x] `re.split(pattern, s)`, `re.escape(s)`, `re.fullmatch(pattern, s)`, `re.subn(pattern, repl, s)` returns `[result, count]`
- [x] `csv.writer(fh)` stateful writer with `.writerow(row)` / `.writerows(rows)` bound to a `with open(...)` handle; `csv.DictReader(lines)` returns `[]map[string]string`; `csv.DictWriter(fh, fields)` with `.writeheader()` / `.writerow(d)` / `.writerows(rows)`
- [x] `pathlib.Path` arithmetic (`p / "sub"`), `.name`, `.parent`, `.suffix`, `.stem`
- [x] `pathlib.Path.iterdir()` (returns `[]*Path`; loop var inherits the Path tag so `for child in p.iterdir(): print(child.name)` works), `.mkdir(parents, exist_ok)`, `.unlink()`, `.glob(pattern)` (shell-style glob via `filepath.Glob`), `.rglob(pattern)` (recursive walk via `filepath.WalkDir`; basename-matched against the pattern), `.read_bytes()` / `.write_bytes(s)` (alias the text counterparts since gopy maps `bytes` to `string`), `.match(pattern)` (right-anchored fnmatch on basename; multi-segment patterns match the joined path), `Path.cwd()` / `Path.home()` classmethods returning a freshly-tagged Path from `os.Getwd` / `os.UserHomeDir`
- [x] `json.dumps(v, indent=N)` pretty-prints with N-space indentation; default form keeps the existing Python-style separators
- [x] `json.load(fh)` / `json.dump(v, fh)` for `with open(...) as fh:` handles
- [x] File iteration line by line (`for line in fh:`)
- [x] `datetime.timedelta` keyword constructor with full parameter set (`days`, `seconds`, `microseconds`, `milliseconds`, `minutes`, `hours`, `weeks`)
- [x] `datetime.datetime.strptime(s, fmt)` and `.strftime(fmt)` / `date.strftime(fmt)` (Python format codes `%Y/%m/%d/%H/%M/%S/%y/%B/%b/%A/%a/%p/%j/%z` mapped to Go's reference-time layout)
- [x] `datetime.datetime.fromtimestamp(ts)`, `.fromisoformat(s)` (accepts trailing `Z`, `+HHMM` or `+HH:MM` offset suffixes; the offset validates the form but the parsed datetime keeps the local clock components, matching CPython's `strftime` behavior on tz-aware values), `.utcnow()`, `.weekday()`, `.isoweekday()`, `.timestamp()`, `.replace(year=..., ...)`, `datetime.combine(date, time)`; `date.replace(year=..., month=..., day=...)`
- [x] `timedelta.total_seconds()`, `.days`, `.seconds`, `abs(td)`, `td + td`, `td - td`, `td * int`, `int * td`, `td / int`
- [x] `enum.auto()` (sequential integer assignment, mixes with explicit values). `Enum`, `IntEnum`, `Flag`, `IntFlag`, `StrEnum` bases all collapse to a typed `int64` alias + constants; bitwise ops on `IntFlag` work via Go's `& | ^` over the alias type
- [x] `@dataclass` field defaults via `field(default=...)` / `field(default_factory=list/dict/set)` — fresh container per instance
- [x] `dataclasses.asdict(obj)` (returns `map[string]any`), `dataclasses.astuple(obj)` (returns `[]any`), `dataclasses.replace(obj, **kwargs)` (fresh instance via constructor), `dataclasses.fields(cls_or_obj)` (returns `[]string` of field names)
- [x] `dict.items()` standalone (returns `[]struct{Key, Value}`); for-loop tuple-unpack form unchanged
- [x] `sum(xs, start)` 2-arg form (int/float; promotes to float when either side is float)
- [x] `datetime.date(y, m, d)` / `datetime.time(h, m, s)` with `.year/.month/.day` (`.hour/.minute/.second`), `.isoformat()`, `.weekday()`, `.isoweekday()`; `date.today()`, `date.fromisoformat(s)`
- [x] `subprocess.run` returning a typed `CompletedProcess` (kwargs ignored)
- [x] `argparse` minimal subset: `ArgumentParser()` + `parser.add_argument(...)` + `parser.parse_args([argv])` → `__ArgNamespace`. Reads `--key value` / `--key=value` / short `-k value` / positional args; auto-parses int-shaped values. Honored kwargs on `add_argument`: `type=int` / `type=float` / `type=str` / `type=bool` (drives runtime conversion via `strconv`), `default=...` (literal default), `action="store_true"` / `"store_false"` (boolean flag, no value follows), `dest="name"` (override the resolved name). Subparsers, mutually exclusive groups, custom callable `type=` converters, and CPython's attribute access (`ns.name`) are not wired — use `ns.Get("name")` in gopy code
- [x] `logging` levels writing to stderr; the module-level threshold (default `WARNING` = 30) gates `logging.debug` / `info` / `warning` / `error` / `critical` and `logging.basicConfig(level=...)` lowers it. `logging.getLogger(name)` returns a `__Logger` with `.debug`/`.info`/`.warning`/`.error`/`.critical` plus `.setLevel(level)` / `.getEffectiveLevel()` / `.isEnabledFor(level)` for per-logger overrides. Handlers / formatters / filter chains aren't wired — every emit goes through the default `LEVEL:name:msg` stderr line
- [x] `collections.Counter`
- [x] `collections.defaultdict` (annotation-driven; factory ignored)
- [x] `collections.deque` (list-backed; append/appendleft/pop/popleft)
- [x] `collections.OrderedDict` (annotation-driven; treated as a plain `dict[K, V]` since Python 3.7+ already preserves insertion order)
- [x] `collections.namedtuple("Name", [...])` at module level — synthesizes a struct with `any`-typed fields and a `Name(args)` constructor; accepts both list-of-names and space-separated string forms
- [x] `itertools.chain`, `itertools.chain.from_iterable`, `itertools.accumulate`, `itertools.takewhile`, `itertools.dropwhile`, `itertools.combinations(xs, r)` (any positive `r` literal — emits an r-deep loop nest), `itertools.permutations` (r=2), `itertools.product(*iters)` (N iterables of the same element type), `itertools.islice(it, [start,] stop[, step])`, `itertools.repeat(value, n)` (bounded form), `itertools.starmap(lambda, pairs)`, `itertools.filterfalse(lambda, xs)`, `itertools.compress(data, selectors)`, `itertools.count(start, step, n)` (bounded form), `itertools.zip_longest(a, b[, fillvalue=...])`, `itertools.pairwise(xs)` → list of `(x[i], x[i+1])` pairs, `itertools.batched(xs, n)` → list of n-sized chunks (last may be shorter)
- [x] `itertools.groupby` (with optional `key=` lambda; consecutive grouping like CPython)
- [x] full-arity `itertools.combinations(xs, r)` (positive int literal `r`) and `itertools.product(*iters)` (any number of same-typed iterables) — both lower to an unrolled loop nest at the call site rather than recursing at runtime
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
- [x] `urllib.request.urlopen(url)` HTTP GET returning an `HTTPResponse` analog (`.read()` / `.status` / `.headers` / `.getcode()` / `.close()`); `http.client.HTTPConnection` / `HTTPSConnection` with `.request(...)` / `.getresponse()`. `http.server`'s `BaseHTTPRequestHandler` is registered as a stub — real request handling lives outside the gopy shim layer (write a Go-side `net/http` handler if needed)

### Builtins

- [x] `map(fn, xs)` and `filter(fn, xs)` — when `fn` is an inline lambda
- [x] `zip(a, b)` and `enumerate(xs[, start])` in `for` loops
- [x] `zip(a, b)` / `enumerate(xs[, start])` returning standalone iterables — both materialize as `[][]any` (list of 2-elt pair slices) so they can be passed around as values. Tuple-unpacking inside `for` still uses the optimized native range path
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

- [x] Single-shot CLI that takes a `.py` and writes a built binary (`gopy build script.py -o script`) — transpiles into a temp directory, drops a minimal `go.mod`, and runs `go build` to produce a native executable in one step
- [x] Project mode: `gopy build <dir>` transpiles every `.py` in the directory into a shared Go package, drops a minimal `go.mod`, and runs `go build`. The binary name comes from `-o` (preferred), then `pyproject.toml`'s `[project].name`, then the directory's basename. `requirements.txt` is intentionally ignored — Python libraries must be vendored as `.py` files alongside the application
- [x] Watch mode (`gopy watch [-interval 500ms] input.py`) — polls the input mtime and re-runs `gopy build` on each change. Build failures land on stderr; the watcher keeps going so a bad save doesn't kill the loop
- [x] Source maps / line directives: every emitted `func` is preceded by a `//line <module>.py:<N>` directive taken from the originating `def`, so Go panic stacks now report `script.py:6` rather than the generated Go file's line. Wired up by `transpile.Options.SourceModule` and threaded through `gopy`, `gopy build`, and `gopy-build`
- [ ] LSP / editor diagnostics: report unsupported features at edit time
- [x] Transpile errors prefix the offending module / filename in front of the existing `line N: ...` context, so multi-file builds point at the source instead of just a line number. Full caret-with-source-excerpt rendering is still a future improvement
- [x] CI workflow in this repo (GitHub Actions) running the fixture suite on `ubuntu-latest` + `macos-latest`
- [ ] Continuous benchmarks dashboard so regressions surface in PRs

### Codegen quality

- [ ] Pluggable target packages: emit into multiple Go files / subpackages to mirror Python module layout
- [x] Avoid emitting unused `_ = args` / `_ = kwargs` stubs when the variable is actually used — the codegen now scans the function body for references to the vararg / kwarg name and only emits the silencer when no use was found
- [x] Reuse helper functions across the program: multi-file projects (`gopy build <dir>` and `gopy-build`) now collect every inline runtime helper across translations, emit them once into a shared `gopy_runtime.go`, and prune unused imports from per-file outputs. Single-file builds still inline helpers at the bottom of the single output
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
- [x] Python-style `print()` / `repr()` formatting of containers: lists print as `[1, 2, 3]` (comma-separated, brackets), dicts as `{'k': v, ...}`, strings inside containers use single-quote repr matching CPython, floats keep trailing `.0`, nested containers recurse. Falls back to `reflect` for arbitrary `[][]T` / `map[K]V` shapes
- [x] `repr(obj)` / f-string `!r` dispatch to a user class's `__repr__` via a `Repr() string` interface check, so custom representations match CPython
- [x] `sorted(xs, key=name)` accepts a bare callable (`abs`, `len`, or any user function) in addition to inline lambdas
- [x] `enumerate(xs, start=N)` accepts the `start` keyword (was previously only positional)
- [x] `Optional[Class]` narrows to `*Class` so a field annotated `next: Optional["Node"] = None` round-trips through Go's nil-able pointer types — attribute access on the recovered concrete class keeps its declared shape
- [x] `self.attr: T = default` inside `__init__` now records `T` (not the default value's narrower type) on the struct field, so `None` defaults don't collapse `Optional[T]` annotations to bare `None`
- [x] Bare `list` / `dict` / `tuple` / `set` annotations accepted as `list[Any]` / `dict[Any, Any]` / etc., matching CPython's stdlib type hints
- [x] Module-level `xs: list[T] = []` / `d: dict[K,V] = {}` emit the declared element type for the empty literal instead of `[]any{}`, so later assignments typecheck
- [x] Negative literal index on a list-typed receiver: `xs[-1]` / `self.field[-1]` rewrites to `xs[len(xs)-1]` at codegen time (Go forbids negative constant indices)
- [x] User-class field type resolution: `self.field` access propagates the field's declared type so downstream slicing / method calls typecheck without needing a local binding
- [x] `T = TypeVar("T")` at module scope: subsequent function signatures referencing `T` (params, return, list/dict/tuple/Optional element) lower to Go generic functions (`func f[T any](...) T`). PEP 695 `def f[T](...)` syntax was already supported; this lets classic-style code work too. `ParamSpec` / `TypeVarTuple` declarations are accepted but ignored
- [x] Keyword-only parameters: `def f(*, port=80, scheme="http")` (and the mixed form `def f(a, *, b=1)`) accepted. Defaults are evaluated at call site like regular kwargs; the keyword-only enforcement isn't strict (positional passes through), matching Python's tolerance in practice
- [x] `str.rfind(sub)` / `str.index(sub)` / `str.rindex(sub)` — `index` and `rindex` raise `ValueError` on miss, matching CPython
- [x] `dict.get(key)` single-argument form returns the value-type's zero (or `nil` for `any`) when the key is missing, matching Python's None semantics
- [x] `int(s, base)` parses string `s` with the given radix via `strconv.ParseInt`
- [x] `int(literal_float)` truncates through an IIFE so Go's untyped-constant rule doesn't reject the conversion
- [x] Binary `**` (power): floats route through `math.Pow`; int exponent uses an inline loop so the result stays `int64` like CPython
- [x] Tuple unpack from a list-typed variable: `a, b = pair` (where `pair: list[int]`) destructures via an index temp `__multi_N`
- [x] Star unpack in list literals: `[*xs, 99]` / `[0, *ys]` lower to an IIFE that appends through `[]T{}` so the element type matches each spread source
- [x] `@staticmethod` on classes: emits a free `<Class>_<method>` Go function with no receiver. Call site `Class.method(...)` dispatches identically to `@classmethod`
- [x] `except UserClass as e:` type-asserts `e` to `*UserClass` (instead of leaving it as `any`), so field access on the bound name typechecks
- [x] Enum `.value` accessor: `Color.RED.value` lowers to `int64(ColorRED)`; a variable typed as the enum also supports `.value`
- [x] `str.partition` / `str.rpartition` / `str.split` / `str.rsplit` / `str.splitlines` are now recognized as slice-returning by the multi-assign code, so `head, sep, tail = s.partition("@")` destructures via the index temp
- [x] List / dict comprehensions iterating `range(...)` emit a C-style `for i := lo; i < hi; i += step` loop instead of `for _ := range range(...)` (which was invalid Go)
- [x] f-string float formatting matches CPython's `repr(float)`: whole-valued floats keep the trailing `.0`, so `f"{3.0}"` prints `3.0` rather than Go's default `3`
- [x] Format spec `#` (alternate) and `+` (force-sign) flags. `f"{255:#x}"` → `0xff`, `f"{42:+d}"` → `+42`. Routes through Go's `%#x` / `%+d` verbs in the same shared spec helper as `str.format` / f-strings
- [x] List literal with declared user-class element type (`xs: list[Box] = [Box(1), Box(2)]`) emits `[]*Box{...}` instead of falling back to `[]any{}` — previously only interface-shaped element classes specialized like this
- [x] Dict iteration `for k, v in d.items():` now declares blank `_ = k; _ = v` silencers so the body can reference either name (or neither) without tripping Go's unused-variable check — `_` named loop targets still pass through untouched
- [x] `itertools.chain(a, b, c, ...)` accepts any number of list arguments (was hard-coded to 2)
- [x] @staticmethod and bare `@property` accept on a class without `self`, and `Class.method(...)` rewrites to `Class_method(...)` like @classmethod (covered earlier; see [tests/fixtures/static_class_mix.py](tests/fixtures/static_class_mix.py))
- [x] String iteration: `for c in "abc":` binds `c` as a single-char `string` (matching CPython's per-codepoint iteration). `list("abc")` returns `["a", "b", "c"]` — splits the string on rune boundaries through an IIFE that collects each `string(r)` into `[]string`
- [x] Walrus operator (`:=`) inside list / dict / set / generator comprehensions: any `(y := value)` appearing in the filter or element expression is hoisted to a plain assignment statement at the top of the loop body so `y` is in scope for both the condition and the element
- [x] `isinstance(x, int)` / `isinstance(x, float)` / `isinstance(x, list)` / `isinstance(x, dict)` now work on `any`-typed values via a type switch over the canonical numeric widths (int / int8…64 / float32 / float64) and a `reflect.Kind` check for container shapes. Fixes a long-standing bug where the narrowing form `if isinstance(x, str)` shadowed `x` in the else branch with the zero value of the failed assertion (broke chained `elif isinstance(...)` ladders)
- [x] `any`-typed list literals (`xs: list = [42, "a", 3.14]`) now box numeric elements through `int64(...)` / `float64(...)` so `.(int64)` / `.(float64)` type assertions hit the canonical Python widths rather than Go's untyped-int default
- [x] Set operations on `set[T]` (which gopy lowers to `[]T`): `a & b` (intersection), `a | b` (union), `a - b` (difference), `a ^ b` (symmetric difference) emit IIFEs that build the result slice with map-based de-duplication. Membership (`in` / `not in`) was already supported; `&`/`|` on dicts keeps its existing semantics
- [x] **Class variables**: a class-level annotated field with a default value (`count: int = 0`) that is never assigned via `self.<name>` in any method is hoisted to a module-level `var <Class>_<name> <Ty> = <Default>`. `Class.field` reads/writes and instance reads (`c.field`) route to the hoisted slot, matching Python's class-shared semantics. Per-instance shadowing via `self.field = ...` is detected at lower time and keeps the field as a struct member instead
- [x] Star-splat with mixed positional args: `f(10, 20, *xs)` (and any combination of pre / mid / post splats) builds the vararg slice through an IIFE that appends each piece in source order — previously only the single-splat shape `f(*xs)` was lowered
- [x] `enumerate(s)` over a string binds the value as a single-char `string` (was a rune-int). Indices remain the codepoint position, matching CPython
- [x] `list(xs)` over a list-typed value produces a fresh copy (`append([]T{}, xs...)`) so callers can mutate without aliasing the source. `list(range(...))` materializes the integer sequence into a concrete `[]int64`
- [x] `range(...)` with a negative literal step now compares with `>` instead of `<`, so `range(10, 0, -2)` actually iterates (was skipping the body entirely)
- [x] `Exception.String()` / `.Error()` strip the leading `"ClassName: "` metadata prefix so `print(e)` and `f"{e}"` match CPython's `str(exc)` (which only shows the message). The prefix stays internally so prefix-based `except ClassName` dispatch keeps working
- [x] `assert cond, msg` now panics with an `"AssertionError: <msg>"` prefix so `except AssertionError` can catch it; display strips the prefix the same way as built-in exceptions
- [x] List-builtin argument inference (`sum`, `sorted`, `min`, `max`, `any`, `all`, `reversed`, `map`, `filter`, `chain`, `accumulate`, etc.) now widens through `effectiveType`, so receivers like `self.items` propagate their declared list element type rather than tripping `argument must be a typed list`

### Hard / open questions

- [ ] Runtime model that supports both static Go performance and Python-style dynamic typing where unavoidable (`any` fallback with type-switched dispatch)
- [ ] Memory model: when can we use values vs. pointers? When can we stack-allocate?
- [ ] Concurrency model: should generators become bounded channels by default? How do we surface goroutine leaks?
- [ ] Garbage collection: how to convey that long-lived Python globals become package-level Go vars without leaking goroutines from generators
- [ ] Multi-file project shape: per-package vs. flat-namespace tradeoffs

## License

MIT — see [LICENSE](LICENSE).
