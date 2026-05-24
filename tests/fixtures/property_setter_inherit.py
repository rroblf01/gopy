class Base:
    def __init__(self, size: int) -> None:
        self._size: int = size

    @property
    def size(self) -> int:
        return self._size

    @size.setter
    def size(self, value: int) -> None:
        if value < 0:
            value = 0
        self._size = value


class Child(Base):
    def __init__(self, size: int, label: str) -> None:
        super().__init__(size)
        self.label: str = label


def main() -> None:
    c = Child(5, "x")
    print(c.size)
    c.size = -3  # should route through Base.SetSize -> clamps to 0
    print(c.size)
    c.size = 7
    print(c.size)


if __name__ == "__main__":
    main()
