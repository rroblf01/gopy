from typing import Callable, Any


def trace(fn: Callable[..., Any]) -> Callable[..., Any]:
    return fn


def label(fn: Callable[..., Any]) -> Callable[..., Any]:
    return fn


@trace
def add(a: int, b: int) -> int:
    return a + b


@trace
@label
def mul(a: int, b: int) -> int:
    return a * b


class Counter:
    @trace
    def total(self, xs: list[int]) -> int:
        s: int = 0
        for x in xs:
            s += x
        return s


def main() -> None:
    print(add(2, 3))
    print(mul(2, 3))
    c = Counter()
    print(c.total([1, 2, 3, 4]))


if __name__ == "__main__":
    main()
