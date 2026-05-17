class Counter:
    def __init__(self) -> None:
        self.n = 0

    def inc(self, by: int) -> None:
        self.n += by

    def value(self) -> int:
        return self.n


def main() -> None:
    c: Counter = Counter()
    for i in range(1, 500001):
        c.inc(i)
    print(c.value())


if __name__ == "__main__":
    main()
