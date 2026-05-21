class Counter:
    def __init__(self, n: int) -> None:
        self.n = n

    def __add__(self, k: int) -> "Counter":
        return Counter(self.n + k * 10)

    def __iadd__(self, k: int) -> "Counter":
        return Counter(self.n + k)

    def __isub__(self, k: int) -> "Counter":
        return Counter(self.n - k)


def main() -> None:
    c = Counter(100)
    c += 5
    print(c.n)
    c -= 3
    print(c.n)
    d = c + 2
    print(d.n)


if __name__ == "__main__":
    main()
