from typing import TypeVar

T = TypeVar("T")


def first(xs: list[T]) -> T:
    return xs[0]


def last(xs: list[T]) -> T:
    return xs[-1]


def main() -> None:
    ints: list[int] = [1, 2, 3]
    strs: list[str] = ["a", "b", "c"]
    print(first(ints))
    print(last(ints))
    print(first(strs))
    print(last(strs))


if __name__ == "__main__":
    main()
