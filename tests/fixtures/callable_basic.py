class Box:
    def __init__(self, x: int) -> None:
        self.x = x


def add(a: int, b: int) -> int:
    return a + b


def main() -> None:
    print(callable(add))
    print(callable(Box))
    print(callable(42))
    print(callable("hi"))
    print(callable(None))
    print(callable([1, 2]))


if __name__ == "__main__":
    main()
