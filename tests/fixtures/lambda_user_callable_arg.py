from typing import Callable


def apply_int(fn: Callable[[int], int], x: int) -> int:
    return fn(x)


def reduce_pair(fn: Callable[[int, int], int], a: int, b: int) -> int:
    return fn(a, b)


def transform_str(fn: Callable[[str], str], s: str) -> str:
    return fn(s)


def filter_count(pred: Callable[[int], bool], xs: list[int]) -> int:
    n: int = 0
    for x in xs:
        if pred(x):
            n += 1
    return n


def main() -> None:
    print(apply_int(lambda x: x * 3, 7))
    print(reduce_pair(lambda a, b: a + b * 2, 5, 10))
    print(transform_str(lambda s: s.upper(), "hi"))
    print(filter_count(lambda n: n % 2 == 0, [1, 2, 3, 4, 5, 6]))

    # Keyword passing.
    print(apply_int(x=4, fn=lambda x: x + 100))


if __name__ == "__main__":
    main()
