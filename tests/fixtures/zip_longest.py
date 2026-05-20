from itertools import zip_longest


def main() -> None:
    for pair in zip_longest([1, 2, 3], [10, 20], fillvalue=-1):
        print(pair[0])
        print(pair[1])
    print("---")
    for pair in zip_longest([1, 2], [10, 20, 30, 40], fillvalue=0):
        print(pair[0])
        print(pair[1])


if __name__ == "__main__":
    main()
