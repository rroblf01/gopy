from typing import Protocol


class Comparable(Protocol):
    def cmp(self, other: int) -> int:
        ...


class Container:
    def __init__(self, item: int) -> None:
        self.item = item

    def cmp(self, other: int) -> int:
        return self.item - other


def main() -> None:
    c = Container(10)
    print(c.cmp(3))
    print(c.cmp(15))


if __name__ == "__main__":
    main()
