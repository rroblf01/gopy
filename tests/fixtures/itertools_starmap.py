from itertools import starmap


def main() -> None:
    pairs: list[list[int]] = [[2, 3], [4, 5], [10, 1]]
    for v in starmap(lambda a, b: a + b, pairs):
        print(v)
    print("---")
    for v in starmap(lambda a, b: a * b, pairs):
        print(v)


if __name__ == "__main__":
    main()
