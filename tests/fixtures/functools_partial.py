import functools


def add(a: int, b: int) -> int:
    return a + b


def main() -> None:
    add5 = functools.partial(add, 5)
    r = add5(3)
    print(r)


if __name__ == "__main__":
    main()
