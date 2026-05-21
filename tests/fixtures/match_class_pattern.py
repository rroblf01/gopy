class Point:
    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y


class Circle:
    def __init__(self, r: int) -> None:
        self.r = r


def classify(shape: object) -> str:
    match shape:
        case Point(x=0, y=0):
            return "origin"
        case Point(x=0):
            return "y-axis"
        case Point(y=0):
            return "x-axis"
        case Point():
            return "point"
        case Circle(r=1):
            return "unit-circle"
        case Circle():
            return "circle"
        case _:
            return "other"


def main() -> None:
    print(classify(Point(0, 0)))
    print(classify(Point(0, 5)))
    print(classify(Point(5, 0)))
    print(classify(Point(3, 4)))
    print(classify(Circle(1)))
    print(classify(Circle(5)))


if __name__ == "__main__":
    main()
