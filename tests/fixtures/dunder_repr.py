class Coord:
    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y

    def __repr__(self) -> str:
        return f"Coord({self.x}, {self.y})"

    def __str__(self) -> str:
        return f"({self.x},{self.y})"


def main() -> None:
    p = Coord(3, 4)
    print(p)
    print(repr(p))
    print(str(p))
    print(f"{p!r}")
    print(f"{p}")


if __name__ == "__main__":
    main()
