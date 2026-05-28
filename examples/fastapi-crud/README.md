# FastAPI CRUD → pure-Go binary (gopy demo)

A small FastAPI CRUD that gopy transpiles to a **single statically-linked Go
binary** — no Python interpreter, no `libpython`, no `libc`. The web routing,
Pydantic request validation, and the OpenAPI/Swagger docs are all reimplemented
in Go (`-goweb`), so the running service is "solo Go, poca RAM".

Verified: the whole CRUD runs from a **8.8 MB `scratch` image using ~7.5 MB
RAM**.

## What it does

In-memory `Item` store (`{id, name, price}`) over five routes:

| Method | Path               | Purpose                                  |
|--------|--------------------|------------------------------------------|
| GET    | `/`                | health + item count                      |
| GET    | `/items`           | list all                                 |
| GET    | `/items/{item_id}` | fetch one                                |
| POST   | `/items`           | create (Pydantic `ItemIn` validation)    |
| PUT    | `/items/{item_id}` | replace                                  |
| DELETE | `/items/{item_id}` | delete                                   |

Plus `/openapi.json` and `/docs` (Swagger UI), generated in Go from the route
table — same as FastAPI, no Python.

> The store is in-memory by design. A real **pure-Go `sqlite3`** shim
> (CGO-free, via `modernc.org/sqlite`) is in progress so this can move to a
> persistent DB while staying interpreter-free.

## Run with Docker (recommended for demos)

Build context is the **repo root** — the image compiles the transpiler from
source, then transpiles `app.py`:

```bash
# from the gopy repo root
docker build -f examples/fastapi-crud/Dockerfile -t gopy-crud .
docker run --rm -p 8000:8000 gopy-crud
```

```bash
curl localhost:8000/
curl -X POST localhost:8000/items -H 'Content-Type: application/json' \
     -d '{"name":"widget","price":9.99}'
curl localhost:8000/items
curl -X PUT localhost:8000/items/1 -H 'Content-Type: application/json' \
     -d '{"name":"widget-v2","price":12.0}'
curl -X DELETE localhost:8000/items/1
open http://localhost:8000/docs        # Swagger UI
```

## Run without Docker

```bash
# from the repo root, build the transpiler once
CGO_ENABLED=0 go build -o gopy ./cmd/gopy

# transpile + compile the demo to a static binary
CGO_ENABLED=0 ./gopy build -goweb -o app examples/fastapi-crud/app.py
./app          # serves on :8000
```

### Compare against stock CPython FastAPI

To run the *original* `app.py` under CPython (for parity checks) — using `uv`
and a virtual env, nothing system-wide:

```bash
cd examples/fastapi-crud
uv venv .venv
uv pip install --python .venv/bin/python -r requirements.txt
.venv/bin/python app.py
```

## gopy subset notes

These shapes are dictated by gopy's supported subset, not by FastAPI:

- **Entrypoint:** the `if __name__ == "__main__": uvicorn.run(app, port=8000)`
  guard body becomes gopy's synthesized `func main()` (`app.run()` works too).
- **Read-back returns are `Any`:** gopy's `dict` is Go `map[any]any`, so a value
  read back out of the store (`items[item_id]`) is `any`. Handlers that return a
  stored value are annotated `-> Any` so both that and a fresh `dict` literal
  type-check through the same return.
- **Validation** comes from declaring the body as a Pydantic model
  (`class ItemIn(BaseModel)`): a bad field yields a FastAPI-shaped HTTP 422.
