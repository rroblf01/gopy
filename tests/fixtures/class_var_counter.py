class Counter:
    count: int = 0

    def __init__(self, name: str) -> None:
        self.name = name
        Counter.count += 1


def main() -> None:
    Counter("a")
    Counter("b")
    Counter("c")
    print(Counter.count)


if __name__ == "__main__":
    main()
