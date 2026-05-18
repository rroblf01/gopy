from itertools import chain, accumulate


def main() -> None:
    a: list[int] = [1, 2, 3]
    b: list[int] = [4, 5, 6]
    # CPython's chain returns an iterator; the gopy shim returns a list.
    # Iterating works the same either way.
    for v in chain(a, b):
        print(v)
    for v in accumulate(a):
        print(v)


if __name__ == "__main__":
    main()
