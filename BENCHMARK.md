# Benchmarks

Performance and memory measurements for gopy, across its two execution regimes:

1. **Fast path** — pure Python transpiled to Go and compiled. No Python runtime
   at all; the binary is native Go.
2. **Bridge path** — programs that call third-party libraries (e.g. FastAPI)
   run those libraries in an *embedded CPython* interpreter, with only the
   transpiled user code (handler bodies) executing as compiled Go, reached
   through the reverse bridge.

The two regimes have opposite cost profiles, so they are reported separately.
All times are wall-clock; RSS is peak resident set size.

**Environment** (2026-05-27):

| | |
|---|---|
| OS | Linux 6.18.32-2-lts x86_64 |
| Go | go1.26.3-X:nodwarf5 |
| CPython | 3.14.5 |
| FastAPI | 0.136.3 (starlette 1.1.0, pydantic 2.13.4) |

---

## 1. Fast path (pure Go) — CPU-bound

CPython vs. the gopy-transpiled Go binary on identical CPU-bound workloads.
Mean of 7 timed runs after 2 warmup runs (`cmd/gopy-bench -n 7 -warmup 2`).
Lower wall time = faster; lower RSS = less RAM.

| Benchmark   | CPython (ms) | gopy Go (ms) | Speedup | CPython RSS (MB) | gopy RSS (MB) | RSS save |
|-------------|-------------:|-------------:|--------:|-----------------:|--------------:|---------:|
| bench_loop  |       113.67 |         2.00 |  56.7x  |            12.76 |          6.32 |   2.02x  |
| bench_class |        46.71 |         1.56 |  29.9x  |            12.92 |          6.28 |   2.06x  |
| bench_fib   |       135.42 |         5.64 |  24.0x  |            12.78 |          6.47 |   1.98x  |

**Takeaway:** when no Python library is involved, gopy is **24–57× faster** and
uses **~2× less memory**. This is the headline case — a self-contained,
statically-typed Go binary with no interpreter.

Reproduce:

```bash
go run ./cmd/gopy-bench -n 7 -warmup 2 tests/fixtures/bench_loop.py
```

---

## 2. Bridge path (FastAPI) — embedded CPython + Go handlers

A real FastAPI app, driven in-process by `TestClient` (full Starlette ASGI
stack). The framework — routing, validation, JSON — runs in the embedded
CPython; only the handler **bodies** are compiled Go, invoked per request
through the reverse bridge (Python → Go → Python). Compared against the *same*
app under native venv CPython.

Wall time is total for N requests; per-request in parentheses. RSS measured via
`getrusage(RUSAGE_CHILDREN)`.

### Light handlers (framework-dominated) — 3000 requests

| Endpoint           | native CPython | transpiled gopy | ratio       |
|--------------------|---------------:|----------------:|------------:|
| `/noop`            | 3504.8 ms (1.17 ms/req) | 4100.1 ms (1.37 ms/req) | **0.85×** (gopy slower) |
| `/compute/2000`    | 3670.6 ms (1.22 ms/req) | 4177.9 ms (1.39 ms/req) | **0.88×** (gopy slower) |
| peak RSS           | 85.5 MB        | 108.4 MB        | gopy +27%   |

### Heavy handler (compute-dominated) — 300 requests, 200k-iter loop each

| Endpoint           | native CPython | transpiled gopy | ratio       |
|--------------------|---------------:|----------------:|------------:|
| `/heavy/200000`    | 4358.2 ms (14.53 ms/req) | 483.9 ms (1.61 ms/req) | **9.0×** (gopy faster) |
| peak RSS           | 53.5 MB        | 59.1 MB         | gopy +10%   |

**Takeaway — the crossover is the whole story.** On the bridge path the
framework itself gains nothing (it is still interpreted CPython), and each
request pays an extra Python→Go→Python marshaling hop. So:

- **Trivial handlers**: gopy is ~15% *slower* than native CPython — the bridge
  overhead is pure cost when the handler does no real work.
- **Compute-heavy handlers**: the compiled-Go body dominates and gopy wins
  big — **9×** here — easily repaying the bridge cost.
- **Memory**: gopy uses *more* RAM on this path (≈ +10–27%): it carries both
  the Go runtime **and** the full embedded CPython with fastapi/starlette/
  pydantic resident. The fast-path RAM savings do not apply when CPython is
  embedded.

The break-even point is roughly "does the handler do more CPU work than the
~0.2 ms/req bridge marshaling overhead costs". I/O-bound or trivial endpoints
favor native CPython; CPU-bound endpoints favor gopy, increasingly so with load.

Reproduce (needs a `uv` venv with fastapi/httpx; see STATUS.md):

```bash
go run ./cmd/gopy build -bridge -o /tmp/fapi_heavy /tmp/bridgetest/fapi_heavy.py
SP=/tmp/bridgetest/.venv/lib/python3.14/site-packages
PYTHONPATH=$SP /tmp/bridgetest/fapi_heavy            # transpiled gopy
PYTHONPATH=$SP /tmp/bridgetest/.venv/bin/python /tmp/bridgetest/fapi_heavy.py  # native
```

---

## 3. Go-native web runtime (`-goweb`) vs native FastAPI

The third regime: instead of embedding CPython, gopy reimplements the framework
routing in Go (`gopy build -goweb`), so the binary is pure Go / no interpreter.
This compares the *same* route — `GET /compute/{n}` summing a 500-iteration loop
and returning a JSON dict — served over real HTTP by:

- **native FastAPI** under `uvicorn` (1 worker), CPython 3.14;
- **gopy `-goweb`**, a statically-linked Go `net/http` binary.

Load generated with `wrk -t4 -c50 -d6s`. RSS is peak resident of the server
process under load.

| Metric            | native FastAPI/uvicorn | gopy `-goweb` (pure Go) | ratio        |
|-------------------|-----------------------:|------------------------:|-------------:|
| Throughput (req/s)|               3,045    |             185,421     | **60.9× more** |
| Latency (mean)    |              15.73 ms  |               0.34 ms   | **46× lower**  |
| Peak RSS (load)   |              55.4 MB   |              14.9 MB    | **3.7× less**  |
| Idle RSS          |              ~55 MB    |               6.4 MB    | ~8.6× less     |
| Interpreter       |     CPython resident   |        none (pure Go)   | —            |

**Takeaway:** when the framework layer is reimplemented in Go rather than run in
embedded CPython, the result is decisive on *both* axes — ~61× the throughput and
~3.7× less RAM under load — because there is no interpreter, no ASGI/Python per
request, just compiled Go on `net/http`. This is the regime that actually
achieves the "solo Go, poca RAM" goal for a web service. The cost is scope: only
the framework's routing/handler layer is reimplemented (FastAPI shape today);
handlers that need a Python C/Rust extension (pydantic-core validation, numpy)
still require the bridge or a Go-native equivalent.

**Fairness note:** uvicorn ran with a single worker (one CPU core); it scales with
`--workers N` across cores, whereas the Go server uses all cores natively
(`GOMAXPROCS`). Even normalized per-core, Go is far ahead, but the headline ratio
reflects 1-worker uvicorn vs all-core Go. The handler is light (500-iter loop);
the gap is dominated by interpreter/framework per-request overhead, which is
representative of typical small API handlers.

Reproduce:

```bash
CGO_ENABLED=0 go run ./cmd/gopy build -goweb -o /tmp/bench_web /tmp/bridgetest/bench_web.py
/tmp/bench_web &                                   # pure-Go server on :8002
SP=/tmp/bridgetest/.venv/lib/python3.14/site-packages
PYTHONPATH=$SP /tmp/bridgetest/.venv/bin/python -m uvicorn bench_native:app --port 8001 &
wrk -t4 -c50 -d6s http://localhost:8002/compute/500   # goweb
wrk -t4 -c50 -d6s http://localhost:8001/compute/500   # native FastAPI
```

---

## Notes & caveats

- Bench wall times exclude process startup where the harness allows it
  (`cmd/gopy-bench`); the FastAPI numbers include a single warmup request but
  the timed loop excludes interpreter/app startup.
- The bridge marshaling cost scales with argument/return complexity. These
  handlers use scalar params and small dict returns; larger payloads cost more.
- Go maps are unordered, so a dict-returning handler may emit JSON keys in a
  different order than CPython — values are identical (see STATUS.md caveats).
- Numbers are single-machine, single-run-set; treat ratios as order-of-magnitude
  guidance, not precise constants.
