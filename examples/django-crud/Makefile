# uv-driven build for the gopy Django CRUD demo.
#
# Hybrid model: routing in pure Go (-goweb maps Django's `urlpatterns` onto
# net/http), the ORM + sqlite3 in the embedded CPython interpreter (-bridge).
# No uvicorn / daphne / gunicorn — the Go binary IS the HTTP server.
#
#   make venv      # uv venv + install django into .venv
#   make build     # transpile app.py -> ./app (Go binary, CGO=on for libpython)
#   make run       # run the transpiled binary (:8000)
#   make run-ref   # run the ORIGINAL app under CPython in .venv (parity check)
#   make docker
#   make clean

REPO := ../..
PY   := .venv/bin/python
GOPY := $(REPO)/bin/gopy
SITE := $(shell .venv/bin/python -c "import sysconfig; print(sysconfig.get_path('purelib'))" 2>/dev/null)

.PHONY: venv build run run-ref docker clean

venv:
	uv venv .venv
	uv pip install --python $(PY) -r requirements.txt

$(GOPY):
	cd $(REPO) && CGO_ENABLED=0 go build -o bin/gopy ./cmd/gopy

# CGO ON: the bridge embeds CPython, which needs libpython3.X linked in.
build: $(GOPY)
	$(GOPY) build -goweb -bridge -python $(PY) -o app app.py

run: build
	rm -f /tmp/gopy_django_crud.db
	PYTHONPATH=$(SITE) ./app

run-ref:
	rm -f /tmp/gopy_django_crud.db
	$(PY) app.py

docker:
	cd $(REPO) && docker build -f examples/django-crud/Dockerfile -t gopy-django .
	docker run --rm -p 8000:8000 gopy-django

clean:
	rm -rf .venv app /tmp/gopy_django_crud.db $(GOPY)
