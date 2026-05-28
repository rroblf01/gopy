# Django CRUD → hybrid Go + bridged ORM (gopy demo)

A Django CRUD that gopy transpiles to a hybrid binary: **routing in pure Go**
(gopy's `-goweb` maps Django's `urlpatterns` onto `net/http`), and the
**Django ORM + sqlite3 inside the embedded CPython interpreter** (`-bridge`).

> **No uvicorn / daphne / gunicorn.** The Go binary IS the HTTP server.
> Django's request/response shim is reimplemented natively; nothing async.

## What it does

In-memory store backed by Django's ORM (`Item` model: `name`, `price`),
sqlite3 file at `/tmp/gopy_django_crud.db`. Five routes:

| Method | Path              | Action                          |
|--------|-------------------|---------------------------------|
| GET    | `/`               | health + item count             |
| GET    | `/items/`         | list (uses `Item.objects.all()`)|
| POST   | `/items/`         | create (`Item.objects.create`)  |
| GET    | `/items/{pk}/`    | one (`.filter(pk=).first()`)    |
| PUT    | `/items/{pk}/`    | update (`item.save()`)          |
| DELETE | `/items/{pk}/`    | delete                          |

Verified end-to-end against the same `app.py` run under CPython, then
re-verified via the gopy binary.

## Run with Make (uv)

```bash
make venv     # uv venv + install Django
make build    # transpile + compile (CGO=on for libpython)
make run      # ./app on :8000
```

```bash
curl localhost:8000/
curl -X POST localhost:8000/items/ -H 'Content-Type: application/json' \
     -d '{"name":"widget","price":9.99}'
curl localhost:8000/items/
curl -X PUT localhost:8000/items/1/ -H 'Content-Type: application/json' \
     -d '{"name":"widget-v2","price":12.0}'
curl -X DELETE localhost:8000/items/1/
```

## Run with Docker

```bash
docker build -f examples/django-crud/Dockerfile -t gopy-django .
docker run --rm -p 8000:8000 gopy-django
```

The image carries libpython3 + the venv's Django so the bridge can drive the
ORM at runtime.

## Why a file-backed sqlite (not `:memory:`)

Django opens a fresh connection per request (thread-local); `:memory:`
databases are per-connection, so the table created at startup wouldn't be
visible from the request thread. A file path keeps them in sync.

## gopy subset notes

- `__name__` in value context (`ROOT_URLCONF=__name__`) is not yet
  recognized — use the literal `"__main__"` (gopy runs the entry as
  `__main__`).
- `class Meta: app_label = "auth"` borrows a pre-registered Django app —
  the model isn't actually shipped by auth; the label just satisfies
  Django's "model needs an app" check.
- The model class is bridged (`class Item(models.Model)`): gopy detects
  the framework base, captures the verbatim source, and re-execs it inside
  the embedded interpreter. `Item.objects.create(...)`, `.all()`,
  `.filter(...)` route through the bridge.

## Management commands — the binary doubles as `manage.py`

With `-goweb -bridge` the synthesized `main()` checks `os.Args`. With no
arguments the Go HTTP server runs (the route table from `urlpatterns`); with
any argument it hands `["manage.py"] + os.Args[1:]` to Django's
`execute_from_command_line` via the bridge. Same binary, both modes.

```bash
./app help                              # full subcommand list
./app migrate                           # ORM migrations
./app shell --command "print(1+1)"
./app shell_plus --plain                # django-extensions REPL
./app show_template_tags                # django-extensions
```

`shell_plus` opens a real Django shell with all models, common `django.db`
imports, and `django-extensions`' Django imports pre-loaded:

```
>>> apps: ['contenttypes', 'auth', 'django_extensions']
>>> models: ['ContentType', 'Permission', 'Group', 'User', 'Item']
```

Mechanism (in [transpile/transpile.go](transpile/transpile.go), helper
`__webServeOrCmd`):

1. Inject `urlpatterns = []` into Python's `__main__` so Django's URL
   discovery (shell_plus, show_urls, runserver_plus) doesn't error on the
   gopy-moved route table.
2. Reconfigure `sys.stdout`/`sys.stderr` for line buffering + write-through
   so command output reaches the terminal before SystemExit.
3. Call `execute_from_command_line(["manage.py"] + os.Args[1:])`.
4. Catch `SystemExit` propagated through the bridge (Django exits via
   `sys.exit(code)` on `--help`, argparse failures, clean completion) and
   translate it back into the process's exit code.

## shell_plus auto-import of bridged models

`Item` lives inside the bridge's class-definition namespace (so its
`__module__` is `builtins`). shell_plus tries `from builtins import Item`
and fails. Workaround: `./app shell_plus --plain --dont-load auth`
(skips the borrowed-app's auto-import). The model is still visible via
`apps.get_model("auth", "Item")` or by typing `Item = apps.get_model(...)`
in the shell.

## Limit (honest)

Like the FastAPI bridge demo, this is the hybrid model — the binary needs
`libpython3.X.so` at runtime and `PYTHONPATH` pointed at the venv's
`site-packages` so CPython finds Django. Pure-Go Django (no interpreter at
all) would require reimplementing the ORM in Go — explicitly out of scope.
