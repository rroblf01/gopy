def inner(n: int) -> int:
    i: int = 0
    while i < n:
        yield i
        i += 1


def outer(n: int) -> int:
    yield -1
    yield from inner(n)
    yield 999


def main() -> None:
    for v in outer(3):
        print(v)


if __name__ == "__main__":
    main()
