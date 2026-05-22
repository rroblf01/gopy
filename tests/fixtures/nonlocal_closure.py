def make_adder(base: int) -> int:
    total: int = base

    def add(x: int) -> int:
        nonlocal total
        total += x
        return total

    add(5)
    add(10)
    return total


def main() -> None:
    print(make_adder(100))


if __name__ == "__main__":
    main()
