class Item:
    def __init__(self, name: str, price: float) -> None:
        self.name = name
        self.price = price

    def __str__(self) -> str:
        return f"{self.name}: ${self.price:.2f}"

    def __repr__(self) -> str:
        return f"Item({self.name!r}, {self.price!r})"


def main() -> None:
    a = Item("apple", 1.5)
    print(a)
    print(repr(a))
    items: list[Item] = [Item("a", 1.0), Item("b", 2.5)]
    for i in items:
        print(i)


if __name__ == "__main__":
    main()
