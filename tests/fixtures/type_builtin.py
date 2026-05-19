class Point:
    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y


def main() -> None:
    print(type(1))
    print(type(1.5))
    print(type("hi"))
    print(type(True))
    print(type(None))
    print(type([1, 2, 3]))
    print(type({"a": 1}))
    p = Point(3, 4)
    print(type(p))
    print(type(1).__name__)
    print(type(p).__name__)


if __name__ == "__main__":
    main()
