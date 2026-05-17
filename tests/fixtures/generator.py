def evens(n: int) -> int:
    i: int = 0
    while i < n:
        yield i
        i += 2


def main() -> None:
    total: int = 0
    for v in evens(10):
        total += v
    print(total)


if __name__ == "__main__":
    main()
