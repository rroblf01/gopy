class Counter:
    n: int

    def __init__(self):
        self.n = 0

    def __call__(self, step: int = 1) -> int:
        self.n += step
        return self.n


def main() -> None:
    c = Counter()
    print(c())
    print(c())
    print(c(10))
    print(c())


if __name__ == "__main__":
    main()
