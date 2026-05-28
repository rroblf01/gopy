# gopy Django CRUD demo — hybrid: routing in pure Go, ORM via embedded CPython.
#
# Build context = gopy REPO ROOT:
#
#   docker build -f examples/django-crud/Dockerfile -t gopy-django .
#   docker run --rm -p 8000:8000 gopy-django
#
# Image carries libpython3 + Django (CPython runs the ORM); the Go binary is
# the HTTP server (no uvicorn / daphne / gunicorn).

# --- Stage 1: transpile Python -> Go, compile against libpython ---------------
FROM golang:1.26-bookworm AS build

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv
# libpython3-dev provides the CPython headers + library the bridge links to;
# pkg-config exposes them via python3-embed.pc. We also need a runtime python3
# binary so uv can build its venv against the same minor.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        python3 python3-dev python3-pip libpython3-dev pkg-config build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY . .

# uv venv with Django installed; gopy uses .venv's python.
RUN uv venv --python python3 /opt/venv \
    && uv pip install --python /opt/venv/bin/python -r examples/django-crud/requirements.txt

# Transpiler itself is cgo-free.
RUN CGO_ENABLED=0 go build -o /usr/local/bin/gopy ./cmd/gopy

# Demo build is CGO=on (bridge embeds CPython).
RUN gopy build -goweb -bridge \
    -python /opt/venv/bin/python \
    -dumper scripts/py_ast_dump.py \
    -o /out/app \
    examples/django-crud/app.py

# Drop a tiny launcher that points PYTHONPATH at the venv site-packages so
# the runtime CPython finds Django.
RUN printf '#!/bin/sh\nexec env PYTHONPATH=/opt/venv/lib/python3*/site-packages /app/app\n' > /out/launcher.sh \
    && chmod +x /out/launcher.sh

# --- Stage 2: minimal runtime with libpython + Django -------------------------
FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends libpython3.11 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /opt/venv /opt/venv
COPY --from=build /out/app /app/app
COPY --from=build /out/launcher.sh /app/launcher.sh
EXPOSE 8000
ENTRYPOINT ["/app/launcher.sh"]
