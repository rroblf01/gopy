class Wrap:
    def __init__(self, n: int) -> None:
        self.n = n

    def __pos__(self) -> "Wrap":
        return Wrap(self.n)

    def __neg__(self) -> "Wrap":
        return Wrap(-self.n)

    def __invert__(self) -> "Wrap":
        return Wrap(~self.n)

    def __str__(self) -> str:
        return f"W({self.n})"


def main() -> None:
    w = Wrap(5)
    print(+w)
    print(-w)
    print(~w)


if __name__ == "__main__":
    main()
