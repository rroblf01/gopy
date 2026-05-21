class Bag:
    def __init__(self, n: int) -> None:
        self.n = n

    def __bool__(self) -> bool:
        return self.n > 0

    def __len__(self) -> int:
        return self.n * 2

    def __abs__(self) -> int:
        v = self.n
        if v < 0:
            v = -v
        return v


def main() -> None:
    a = Bag(5)
    b = Bag(0)
    c = Bag(-3)
    print(bool(a))
    print(bool(b))
    print(bool(c))
    print(len(a))
    print(len(b))
    print(abs(c))
    print(abs(a))


if __name__ == "__main__":
    main()
