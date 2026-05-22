class Vector:
    def __init__(self, x: float, y: float) -> None:
        self.x = x
        self.y = y

    def magnitude(self) -> float:
        return (self.x * self.x + self.y * self.y) ** 0.5

    def __add__(self, o: "Vector") -> "Vector":
        return Vector(self.x + o.x, self.y + o.y)

    def __eq__(self, o: "Vector") -> bool:
        return self.x == o.x and self.y == o.y

    def __str__(self) -> str:
        return f"V({self.x}, {self.y})"


def main() -> None:
    a = Vector(3.0, 4.0)
    print(a)
    print(a.magnitude())
    b = Vector(1.0, 2.0)
    c = a + b
    print(c)
    print(a == Vector(3.0, 4.0))
    print(a == b)


if __name__ == "__main__":
    main()
