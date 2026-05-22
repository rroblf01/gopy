from itertools import chain


def main() -> None:
    a: list[int] = [1, 2, 3]
    b: list[int] = [4, 5, 6]
    c: list[int] = [7, 8, 9]
    merged: list[int] = list(chain(a, b, c))
    print(merged)
    print(sum(merged))


if __name__ == "__main__":
    main()
