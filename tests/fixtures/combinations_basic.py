from itertools import combinations, product


def main() -> None:
    xs: list[int] = [1, 2, 3, 4]
    pairs: list[list[int]] = list(combinations(xs, 2))
    print(len(pairs))
    for p in pairs:
        print(p[0])
        print(p[1])

    a: list[int] = [1, 2]
    b: list[int] = [10, 20, 30]
    grid: list[list[int]] = list(product(a, b))
    print(len(grid))
    for row in grid:
        print(row[0] * row[1])


if __name__ == "__main__":
    main()
