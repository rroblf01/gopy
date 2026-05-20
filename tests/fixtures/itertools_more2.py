from itertools import filterfalse, compress


def main() -> None:
    for v in filterfalse(lambda x: x % 2 == 0, [1, 2, 3, 4, 5, 6]):
        print(v)
    print("---")
    for v in compress([10, 20, 30, 40, 50], [1, 0, 1, 0, 1]):
        print(v)


if __name__ == "__main__":
    main()
