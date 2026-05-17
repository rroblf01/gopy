class Point:
    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y

    def move(self, dx: int, dy: int) -> None:
        self.x += dx
        self.y += dy

    def manhattan(self) -> int:
        return self.x + self.y


def main() -> None:
    p: Point = Point(3, 4)
    print(p.manhattan())
    p.move(2, 1)
    print(p.x)
    print(p.y)
    print(p.manhattan())


if __name__ == "__main__":
    main()
