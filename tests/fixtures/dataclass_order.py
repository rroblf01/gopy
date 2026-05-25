from dataclasses import dataclass


@dataclass(order=True)
class Point:
    x: int
    y: int


def main() -> None:
    a = Point(1, 2)
    b = Point(1, 3)
    c = Point(2, 0)
    print(a < b)
    print(a <= b)
    print(b < c)
    print(c > a)
    print(a < a)
    print(a <= a)


if __name__ == "__main__":
    main()
