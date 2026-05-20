class Point:
    def __init__(self, x: int, y: int) -> None:
        self.x = x
        self.y = y

    def __str__(self) -> str:
        return f"Point({self.x},{self.y})"

    def __len__(self) -> int:
        return self.x + self.y


def main() -> None:
    p = Point(3, 4)
    print(p)
    print(p.__str__())
    print(p.__len__())


if __name__ == "__main__":
    main()
