from dataclasses import dataclass


@dataclass
class Point:
    x: int
    y: int


@dataclass
class Circle:
    cx: int
    cy: int
    r: int


def describe(shape: object) -> str:
    match shape:
        case Point(x, y):
            return f"point at {x},{y}"
        case Circle(cx, cy, r):
            return f"circle r={r} at {cx},{cy}"
        case _:
            return "unknown"


def main() -> None:
    print(describe(Point(1, 2)))
    print(describe(Circle(3, 4, 5)))
    print(describe(Point(-1, -1)))
    # mixed: bind a, ignore b
    p = Point(10, 20)
    match p:
        case Point(a, _):
            print(a)


if __name__ == "__main__":
    main()
