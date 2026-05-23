class Bucket:
    items: list[int]

    def __init__(self) -> None:
        self.items = []

    def add(self, x: int) -> None:
        self.items.append(x)

    def total(self) -> int:
        return sum(self.items)


def main() -> None:
    b = Bucket()
    b.add(10)
    b.add(20)
    b.add(30)
    print(b.total())
    print(b.items)


if __name__ == "__main__":
    main()
