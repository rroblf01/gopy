class Box:
    def __init__(self, n: int) -> None:
        self.n = n

    def add(self, k: int) -> "Box":
        self.n += k
        return self

    def mul(self, k: int) -> "Box":
        self.n *= k
        return self


def main() -> None:
    b = Box(5).add(3).mul(2).add(1)
    print(b.n)


if __name__ == "__main__":
    main()
