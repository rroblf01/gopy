class Counter:
    def __init__(self) -> None:
        self.n = 0

    def add(self, k: int) -> None:
        self.n += k


def main() -> None:
    c = Counter()
    c.add(5)
    c.add(10)
    c.n += 100
    print(c.n)


if __name__ == "__main__":
    main()
