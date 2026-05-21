from abc import ABC, abstractmethod


class Shape(ABC):
    @abstractmethod
    def area(self) -> float:
        ...

    @abstractmethod
    def name(self) -> str:
        ...


class Square(Shape):
    def __init__(self, side: float) -> None:
        self.side = side

    def area(self) -> float:
        return self.side * self.side

    def name(self) -> str:
        return "square"


class Circle(Shape):
    def __init__(self, r: float) -> None:
        self.r = r

    def area(self) -> float:
        return 3.14 * self.r * self.r

    def name(self) -> str:
        return "circle"


def main() -> None:
    sq = Square(2.0)
    ci = Circle(1.0)
    print(sq.name(), sq.area())
    print(ci.name(), ci.area())


if __name__ == "__main__":
    main()
