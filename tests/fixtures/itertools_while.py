from itertools import takewhile, dropwhile


def main() -> None:
    xs: list[int] = [1, 2, 3, 10, 1, 2]
    head: list[int] = takewhile(lambda x: x < 5, xs)
    for v in head:
        print(v)
    tail: list[int] = dropwhile(lambda x: x < 5, xs)
    for v in tail:
        print(v)


if __name__ == "__main__":
    main()
