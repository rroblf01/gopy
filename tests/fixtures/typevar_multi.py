from typing import TypeVar

T = TypeVar("T")


def pair(a: T, b: T) -> list[T]:
    return [a, b]


def head_tail(xs: list[T]) -> list[T]:
    return [xs[0], xs[-1]]


def main() -> None:
    print(pair(1, 2))
    print(pair("a", "b"))
    print(head_tail([10, 20, 30, 40]))
    print(head_tail(["one", "two", "three"]))


if __name__ == "__main__":
    main()
