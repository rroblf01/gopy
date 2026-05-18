def evens(n: int) -> int:
    i: int = 0
    while i < n:
        yield i
        i += 2


def main() -> None:
    it = evens(6)
    print(next(it))
    print(next(it))
    print(next(it))
    # default branch when the generator is exhausted.
    print(next(it, -1))


if __name__ == "__main__":
    main()
