def main() -> None:
    class Point:
        x: int
        y: int

        def __init__(self, x: int, y: int):
            self.x = x
            self.y = y

        def dist_sq(self) -> int:
            return self.x * self.x + self.y * self.y

    p = Point(3, 4)
    print(p.x)
    print(p.y)
    print(p.dist_sq())


if __name__ == "__main__":
    main()
