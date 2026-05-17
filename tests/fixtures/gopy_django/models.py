"""Django-flavored ORM shim used by gopy fixtures.

CPython runs this code directly when a fixture is exercised; the gopy
transpiler recognizes the same class shapes (Model + Field declarations)
and emits an equivalent in-memory Go implementation. Keep the two
behaviors aligned — every method here has a Go-side counterpart in
transpile/stdlib.go.
"""


class Field:
    """Common base for all field types. Tracks the field's declared default."""

    def __init__(self, default=None):
        self.default = default


class CharField(Field):
    def __init__(self, max_length=255, default=""):
        super().__init__(default=default)
        self.max_length = max_length


class IntegerField(Field):
    def __init__(self, default=0):
        super().__init__(default=default)


class BooleanField(Field):
    def __init__(self, default=False):
        super().__init__(default=default)


class _Manager:
    """In-memory queryset-like manager. Stores model instances in a list
    and offers Django-style retrieval helpers."""

    def __init__(self, model_class):
        self.model_class = model_class
        self.records = []

    def all(self):
        return list(self.records)

    def filter(self, **kwargs):
        out = []
        for r in self.records:
            ok = True
            for k, v in kwargs.items():
                if getattr(r, k, None) != v:
                    ok = False
                    break
            if ok:
                out.append(r)
        return out

    def get(self, **kwargs):
        hits = self.filter(**kwargs)
        if not hits:
            raise Exception("DoesNotExist")
        if len(hits) > 1:
            raise Exception("MultipleObjectsReturned")
        return hits[0]

    def create(self, **kwargs):
        r = self.model_class(**kwargs)
        r.save()
        return r


class Model:
    """Base class for ORM models. Subclasses declare CharField / IntegerField
    / BooleanField class attributes; instances are constructed with kwargs."""

    def __init_subclass__(cls, **kw):
        super().__init_subclass__(**kw)
        # Each subclass gets its own Manager bound to its own records list.
        cls.objects = _Manager(cls)
        # Collect declared Field names so __init__ knows which kwargs to
        # accept and what defaults to fill from.
        fields = {}
        for k in dir(cls):
            v = getattr(cls, k, None)
            if isinstance(v, Field):
                fields[k] = v
        cls._fields = fields

    def __init__(self, **kwargs):
        for name, field in type(self)._fields.items():
            setattr(self, name, kwargs.get(name, field.default))

    def save(self):
        cls = type(self)
        if self not in cls.objects.records:
            cls.objects.records.append(self)
