from itertools import chain


def main() -> None:
    nested: list[list[int]] = [[1, 2], [3, 4], [5]]
    for v in chain.from_iterable(nested):
        print(v)
    print("---")
    empty: list[list[int]] = []
    for v in chain.from_iterable(empty):
        print(v)
    print("done")


if __name__ == "__main__":
    main()
