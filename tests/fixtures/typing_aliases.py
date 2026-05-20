from typing import Any, Callable, Iterable, Sequence, Set, Tuple


def first(xs: Sequence[int]) -> int:
    return xs[0]


def has_any(items: Iterable[int], n: int) -> bool:
    for v in items:
        if v == n:
            return True
    return False


def echo(v: Any) -> Any:
    return v


def make_pair() -> Tuple[int, str]:
    return 7, "ok"


def unique(xs: Set[int]) -> int:
    return len(xs)


def main() -> None:
    print(first([10, 20, 30]))
    print(has_any([1, 2, 3], 2))
    print(has_any([1, 2, 3], 99))
    print(echo("hi"))
    a, b = make_pair()
    print(a)
    print(b)
    print(unique([1, 2, 3]))


if __name__ == "__main__":
    main()
