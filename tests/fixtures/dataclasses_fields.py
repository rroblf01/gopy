from dataclasses import dataclass, fields


@dataclass
class Item:
    name: str
    qty: int
    price: float


def main() -> None:
    print(len(fields(Item)))
    it = Item("widget", 3, 9.99)
    print(it.name)
    print(len(fields(it)))


if __name__ == "__main__":
    main()
