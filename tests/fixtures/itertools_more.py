from itertools import permutations, islice, repeat


def main() -> None:
    for pair in permutations([1, 2, 3], 2):
        print(pair[0])
        print(pair[1])
    print("---")
    for v in islice([10, 20, 30, 40, 50], 3):
        print(v)
    print("---")
    for v in islice([10, 20, 30, 40, 50], 1, 4):
        print(v)
    print("---")
    for v in islice([10, 20, 30, 40, 50], 0, 5, 2):
        print(v)
    print("---")
    for v in repeat("x", 4):
        print(v)


if __name__ == "__main__":
    main()
