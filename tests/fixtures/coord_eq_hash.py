class Coord:
    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y

    def __eq__(self, o: "Coord") -> bool:
        return self.x == o.x and self.y == o.y

    def __hash__(self) -> int:
        return self.x * 1000 + self.y

    def __ne__(self, o: "Coord") -> bool:
        return not (self.x == o.x and self.y == o.y)


def main() -> None:
    a = Coord(1, 2)
    b = Coord(1, 2)
    c = Coord(3, 4)
    print(a == b)
    print(a == c)
    print(a != c)
    print(hash(a))
    print(hash(c))


if __name__ == "__main__":
    main()
