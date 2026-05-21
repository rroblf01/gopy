import math


class Approx:
    def __init__(self, v: int) -> None:
        self.v = v

    def __round__(self, n: int = 0) -> int:
        if n == 0:
            v: int = self.v
            return ((v + 5) // 10) * 10
        return self.v + n

    def __ceil__(self) -> int:
        return self.v + 100

    def __floor__(self) -> int:
        return self.v - 100

    def __trunc__(self) -> int:
        return self.v


def main() -> None:
    a = Approx(123)
    print(round(a))
    print(round(a, 7))
    print(math.ceil(a))
    print(math.floor(a))
    print(math.trunc(a))


if __name__ == "__main__":
    main()
