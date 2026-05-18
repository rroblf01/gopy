from functools import partial


def add(a: int, b: int) -> int:
    return a + b


def mul3(a: int, b: int, c: int) -> int:
    return a * b * c


def main() -> None:
    add5 = partial(add, 5)
    print(add5(3))
    print(add5(10))
    twice = partial(mul3, 2)
    print(twice(3, 4))
    six = partial(mul3, 2, 3)
    print(six(7))


if __name__ == "__main__":
    main()
