# The @cache / @route decorators below are passthrough — gopy accepts
# them at lower time and skips invoking the body, so the wrapped fn
# still runs as written. CPython does invoke them, so we need real
# identity wrappers underneath; gopy just never reaches that code.

from typing import Any


def _identity(fn: Any) -> Any:
    return fn


def cache(maxsize: int) -> Any:
    return _identity


def route(path: str) -> Any:
    return _identity


@cache(maxsize=128)
@route("/hello")
def handle(name: str) -> str:
    return "hello " + name


class Service:
    @cache(maxsize=64)
    def lookup(self, key: str) -> int:
        return len(key)


def main() -> None:
    print(handle("ana"))
    s = Service()
    print(s.lookup("abcd"))


if __name__ == "__main__":
    main()
