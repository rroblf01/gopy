"""Django CRUD demo — hybrid gopy: web tier in pure Go, ORM via the bridge.

gopy's `-goweb` recognises Django's `urlpatterns` and maps each route onto a
native `net/http` handler; `-bridge` lets the Django ORM (metaclass-driven, not
transpilable) run inside the embedded CPython interpreter. No uvicorn / daphne /
gunicorn — the Go binary IS the server. sqlite3 is CPython's built-in C
module, available through the bridge.

Build + run:
    gopy build -goweb -bridge -venv-deps -python .venv/bin/python app.py
    PYTHONPATH=$(pwd)/.venv/lib/python*/site-packages ./app
    # or: make docker

Then: curl localhost:8000/items/, POST /items/, etc.
"""
from django.conf import settings

settings.configure(
    DEBUG=True,
    DATABASES={
        "default": {
            "ENGINE": "django.db.backends.sqlite3",
            "NAME": "/tmp/gopy_django_crud.db",
        },
    },
    INSTALLED_APPS=[
        "django.contrib.contenttypes",
        "django.contrib.auth",
        "django_extensions",
    ],
    ROOT_URLCONF="__main__",
)

import django
django.setup()

from django.db import models, connection
from django.urls import path
from django.http import JsonResponse, HttpResponse
import json


class Item(models.Model):
    name = models.CharField(max_length=100)
    price = models.FloatField()

    class Meta:
        # Borrow a registered app label; the model isn't actually shipped by
        # auth — this just satisfies Django's "model needs an app" check.
        app_label = "auth"


with connection.schema_editor() as se:
    se.create_model(Item)


def root(request):
    return JsonResponse({"service": "gopy-django-crud", "items": Item.objects.count()})


def list_items(request):
    if request.method == "POST":
        data = json.loads(request.body)
        item = Item.objects.create(name=data["name"], price=data["price"])
        return JsonResponse({"id": item.id, "name": item.name, "price": item.price})
    return JsonResponse({"items": [{"id": i.id, "name": i.name, "price": i.price} for i in Item.objects.all()]})


def item_detail(request, pk: int):
    item = Item.objects.filter(pk=pk).first()
    if item is None:
        return JsonResponse({"error": "not found", "id": pk})
    if request.method == "DELETE":
        item.delete()
        return JsonResponse({"deleted": pk})
    if request.method == "PUT":
        data = json.loads(request.body)
        item.name = data["name"]
        item.price = data["price"]
        item.save()
        return JsonResponse({"id": item.id, "name": item.name, "price": item.price})
    return JsonResponse({"id": item.id, "name": item.name, "price": item.price})


urlpatterns = [
    path("", root),
    path("items/", list_items),
    path("items/<int:pk>/", item_detail),
]


def runserver(port):
    pass


if __name__ == "__main__":
    runserver(8000)
