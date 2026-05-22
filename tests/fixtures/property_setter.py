class Box:
    n: int

    def __init__(self, n: int) -> None:
        self.n = n

    @property
    def value(self) -> int:
        return self.n

    @value.setter
    def value(self, v: int) -> None:
        self.n = v * 2


def main() -> None:
    b = Box(3)
    print(b.value)
    b.value = 10
    print(b.value)
    b.value = 7
    print(b.value)


if __name__ == "__main__":
    main()
