"""FastAPI CRUD demo — transpiled to a pure-Go binary by gopy (`-goweb`).

No Python interpreter at runtime: gopy maps FastAPI routing, Pydantic request
validation, and the OpenAPI/Swagger docs onto a native `net/http` server. The
store is an in-memory dict (sqlite is a separate, in-progress pure-Go shim).

Build + run:
    gopy build -goweb -o app app.py && ./app
    # or: docker build -f examples/fastapi-crud/Dockerfile -t gopy-crud . && docker run -p 8000:8000 gopy-crud

gopy-specific note (the supported subset, not a stock FastAPI quirk):
  * handlers that read a value back out of the store return `Any` — gopy's
    `dict` is `map[any]any`, so a stored value reads back as `any`; annotating
    the return as `Any` lets both that and a fresh dict literal flow through.
"""
from fastapi import FastAPI
from pydantic import BaseModel
from typing import Any
import uvicorn

app = FastAPI()


class ItemIn(BaseModel):
    name: str
    price: float


# In-memory store: id -> item dict, plus an auto-increment counter.
items: dict = {}
next_id: int = 1


@app.get("/")
def root() -> dict:
    return {"service": "gopy-fastapi-crud", "items": len(items)}


@app.get("/items")
def list_items() -> Any:
    result = []
    for k in items:
        result.append(items[k])
    return result


@app.get("/items/{item_id}")
def get_item(item_id: int) -> Any:
    if item_id in items:
        return items[item_id]
    return {"error": "not found", "id": item_id}


@app.post("/items")
def create_item(payload: ItemIn) -> Any:
    global next_id
    record = {"id": next_id, "name": payload.name, "price": payload.price}
    items[next_id] = record
    next_id = next_id + 1
    return record


@app.put("/items/{item_id}")
def update_item(item_id: int, payload: ItemIn) -> Any:
    if item_id in items:
        record = {"id": item_id, "name": payload.name, "price": payload.price}
        items[item_id] = record
        return record
    return {"error": "not found", "id": item_id}


@app.delete("/items/{item_id}")
def delete_item(item_id: int) -> Any:
    if item_id in items:
        del items[item_id]
        return {"deleted": item_id}
    return {"error": "not found", "id": item_id}


if __name__ == "__main__":
    uvicorn.run(app, port=8000)
