from typing import Callable


def apply(fn: Callable[[int], int], x: int) -> int:
    return fn(x)


def double(x: int) -> int:
    return x * 2


def square(x: int) -> int:
    return x * x


def main() -> None:
    print(apply(double, 5))
    print(apply(square, 5))


if __name__ == "__main__":
    main()
