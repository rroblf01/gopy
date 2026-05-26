from itertools import combinations_with_replacement


def main() -> None:
    xs: list[int] = [1, 2, 3]
    pairs: list[list[int]] = list(combinations_with_replacement(xs, 2))
    print(len(pairs))
    for p in pairs:
        print(p[0])
        print(p[1])


if __name__ == "__main__":
    main()
