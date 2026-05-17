class Counter:
    def __init__(self, n: int) -> None:
        self.n = n

    @classmethod
    def zero(cls) -> "Counter":
        return cls(0)

    @classmethod
    def starting_at(cls, n: int) -> "Counter":
        return cls(n)

    def value(self) -> int:
        return self.n


def main() -> None:
    a: Counter = Counter.zero()
    print(a.value())
    b: Counter = Counter.starting_at(42)
    print(b.value())


if __name__ == "__main__":
    main()
