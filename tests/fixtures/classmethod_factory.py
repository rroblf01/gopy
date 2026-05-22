class Counter:
    n: int = 0

    def __init__(self, n: int) -> None:
        self.n = n

    @classmethod
    def zero(cls) -> "Counter":
        return Counter(0)

    @classmethod
    def from_int(cls, n: int) -> "Counter":
        return Counter(n * 2)

    def value(self) -> int:
        return self.n


def main() -> None:
    a = Counter.zero()
    b = Counter.from_int(5)
    print(a.value())
    print(b.value())


if __name__ == "__main__":
    main()
