# uv-driven build for the gopy FastAPI CRUD demo.
#
# The Python side is managed entirely by uv in a local .venv (nothing
# system-wide); gopy then consults that .venv's interpreter to parse the source
# and transpile it to a pure-Go binary.
#
#   make venv      # uv venv + install deps into .venv
#   make build     # transpile app.py -> ./app (static Go binary, via .venv python)
#   make run       # run the transpiled Go binary (:8000)
#   make run-ref   # run the ORIGINAL app under CPython in .venv (parity check)
#   make docker    # build + run the scratch image
#   make clean

REPO := ../..
PY   := .venv/bin/python
GOPY := $(REPO)/bin/gopy

.PHONY: venv build run run-ref docker clean

venv:
	uv venv .venv
	uv pip install --python $(PY) -r requirements.txt

# Build the transpiler once (CGO off: the GoWeb path needs no libpython).
$(GOPY):
	cd $(REPO) && CGO_ENABLED=0 go build -o bin/gopy ./cmd/gopy

# Transpile + compile the demo to a static binary, using the .venv interpreter.
build: $(GOPY)
	CGO_ENABLED=0 $(GOPY) build -goweb -python $(PY) -o app app.py

run: build
	./app

run-ref:
	$(PY) app.py

docker:
	cd $(REPO) && docker build -f examples/fastapi-crud/Dockerfile -t gopy-crud .
	docker run --rm -p 8000:8000 gopy-crud

clean:
	rm -rf .venv app $(GOPY)
