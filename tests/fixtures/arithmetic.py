def add(a: int, b: int) -> int:
    return a + b


def main() -> None:
    x: int = 7
    y: int = 5
    print(add(x, y))
    print(add(x, y) * 2 - 1)


if __name__ == "__main__":
    main()
