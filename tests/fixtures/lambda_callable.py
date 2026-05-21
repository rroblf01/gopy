from typing import Callable


def main() -> None:
    double: Callable[[int], int] = lambda x: x * 2
    print(double(3))
    print(double(10))

    add: Callable[[int, int], int] = lambda a, b: a + b
    print(add(7, 5))

    upper: Callable[[str], str] = lambda s: s.upper()
    print(upper("hello"))

    is_even: Callable[[int], bool] = lambda n: n % 2 == 0
    print(is_even(4))
    print(is_even(5))


if __name__ == "__main__":
    main()
