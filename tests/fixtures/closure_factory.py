from typing import Callable


def make_multiplier(factor: int) -> Callable[[int], int]:
    def mul(x: int) -> int:
        return x * factor
    return mul


def main() -> None:
    double = make_multiplier(2)
    triple = make_multiplier(3)
    print(double(10))
    print(triple(10))
    print(double(triple(5)))


if __name__ == "__main__":
    main()
