from dataclasses import dataclass, is_dataclass


@dataclass
class Point:
    x: int
    y: int


class Plain:
    def __init__(self, n: int) -> None:
        self.n: int = n


def main() -> None:
    print(is_dataclass(Point))
    print(is_dataclass(Plain))

    p: Point = Point(1, 2)
    pl: Plain = Plain(3)
    print(is_dataclass(p))
    print(is_dataclass(pl))


if __name__ == "__main__":
    main()
