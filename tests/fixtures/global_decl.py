total: int = 0


def add(n: int) -> None:
    global total
    total += n


def reset() -> None:
    global total
    total = 0


def main() -> None:
    add(5)
    add(10)
    print(total)
    reset()
    add(7)
    print(total)


if __name__ == "__main__":
    main()
