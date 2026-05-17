class Rect:
    def __init__(self, w: int, h: int) -> None:
        self.w = w
        self.h = h

    @property
    def area(self) -> int:
        return self.w * self.h

    @property
    def perimeter(self) -> int:
        return 2 * (self.w + self.h)


def main() -> None:
    r: Rect = Rect(3, 4)
    print(r.area)
    print(r.perimeter)


if __name__ == "__main__":
    main()
