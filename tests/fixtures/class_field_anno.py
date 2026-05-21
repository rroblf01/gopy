from typing import ClassVar


class Bag:
    items: list[int]
    label: str
    KIND: ClassVar[str] = "container"

    def __init__(self, label: str) -> None:
        self.items = []
        self.label = label

    def add(self, v: int) -> None:
        self.items.append(v)

    def describe(self) -> str:
        return f"{self.label}={len(self.items)}"


def main() -> None:
    b = Bag("first")
    b.add(10)
    b.add(20)
    b.add(30)
    print(b.describe())
    print(len(b.items))


if __name__ == "__main__":
    main()
