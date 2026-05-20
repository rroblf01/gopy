class Vec:
    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y

    def __add__(self, other: "Vec") -> "Vec":
        return Vec(self.x + other.x, self.y + other.y)

    def __sub__(self, other: "Vec") -> "Vec":
        return Vec(self.x - other.x, self.y - other.y)

    def __mul__(self, k: int) -> "Vec":
        return Vec(self.x * k, self.y * k)

    def __lt__(self, other: "Vec") -> bool:
        return self.x * self.x + self.y * self.y < other.x * other.x + other.y * other.y


def main() -> None:
    a = Vec(1, 2)
    b = Vec(3, 4)
    c = a + b
    print(c.x)
    print(c.y)
    d = b - a
    print(d.x)
    print(d.y)
    e = a * 3
    print(e.x)
    print(e.y)
    print(a < b)
    print(b < a)


if __name__ == "__main__":
    main()
