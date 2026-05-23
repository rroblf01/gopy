class Registry:
    items: list = []
    count: int = 0

    @classmethod
    def add(cls, item: str) -> None:
        cls.items.append(item)
        cls.count += 1


def main() -> None:
    Registry.add("a")
    Registry.add("b")
    Registry.add("c")
    print(Registry.count)
    print(Registry.items)


if __name__ == "__main__":
    main()
