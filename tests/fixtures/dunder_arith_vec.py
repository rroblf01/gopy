class Vec:
    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y

    def __add__(self, o: "Vec") -> "Vec":
        return Vec(self.x + o.x, self.y + o.y)

    def __sub__(self, o: "Vec") -> "Vec":
        return Vec(self.x - o.x, self.y - o.y)

    def __mul__(self, k: int) -> "Vec":
        return Vec(self.x * k, self.y * k)

    def __neg__(self) -> "Vec":
        return Vec(-self.x, -self.y)

    def __str__(self) -> str:
        return f"V({self.x},{self.y})"


def main() -> None:
    a = Vec(1, 2)
    b = Vec(3, 4)
    print(a + b)
    print(a - b)
    print(a * 3)
    print(-a)


if __name__ == "__main__":
    main()
