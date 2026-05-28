# gopy FastAPI CRUD demo — transpiled to a pure-Go static binary, run on scratch.
#
# Build context is the gopy REPO ROOT (the Dockerfile needs the transpiler source):
#
#   docker build -f examples/fastapi-crud/Dockerfile -t gopy-crud .
#   docker run --rm -p 8000:8000 gopy-crud
#
# Then: curl localhost:8000/items, /docs, /openapi.json
#
# The final image carries NO Python interpreter and NO libpython — just the
# ~9 MB statically-linked Go binary gopy produced from app.py.

# --- Stage 1: transpile Python -> Go, compile a static binary ----------------
FROM golang:1.26-bookworm AS build

# uv manages the Python side. gopy consults the uv-created .venv's interpreter to
# parse + transpile the source — it only needs CPython's stdlib `ast`, so the
# app's dependencies (fastapi/pydantic/uvicorn) are NOT installed here: gopy
# reimplements their behavior in Go (`-goweb`), it never runs them. (Install
# them with `uv pip install -r requirements.txt` only to run the original under
# CPython for a parity check — see the Makefile's `run-ref`.)
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY . .

# uv-created virtual env; gopy is pointed at its interpreter below.
RUN uv venv --python python3 /opt/venv

# CGO off everywhere: gopy's `bridge` package is cgo-gated (needs libpython /
# python3-embed), but the GoWeb path doesn't use it. Disabling cgo drops that
# build constraint, so neither the transpiler nor the demo needs libpython, and
# the demo binary links nothing — a fully static executable that runs on scratch.
ENV CGO_ENABLED=0

# Build the transpiler itself.
RUN go build -o /usr/local/bin/gopy ./cmd/gopy

# Transpile + compile the demo to a static Go binary, consulting the uv .venv.
RUN gopy build -goweb \
    -python /opt/venv/bin/python \
    -dumper scripts/py_ast_dump.py \
    -o /out/app \
    examples/fastapi-crud/app.py

# --- Stage 2: minimal runtime, zero interpreter ------------------------------
FROM scratch
COPY --from=build /out/app /app
EXPOSE 8000
ENTRYPOINT ["/app"]
