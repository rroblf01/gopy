from dataclasses import dataclass


@dataclass
class Point:
    x: int
    y: int


@dataclass
class Rect:
    tl: Point
    br: Point


def main() -> None:
    p = Point(3, 4)
    print(p.x, p.y)
    r = Rect(Point(0, 0), Point(10, 20))
    print(r.tl.x, r.br.y)


if __name__ == "__main__":
    main()
