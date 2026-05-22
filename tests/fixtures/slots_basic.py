class Point:
    __slots__ = ("x", "y")

    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y


def main() -> None:
    p = Point(3, 4)
    print(p.x)
    print(p.y)
    print(p.x + p.y)


if __name__ == "__main__":
    main()
