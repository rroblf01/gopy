from itertools import groupby


def main() -> None:
    xs: list[int] = [1, 1, 2, 2, 2, 3, 1, 1]
    for k, group in groupby(xs):
        count = 0
        for _ in group:
            count = count + 1
        print(k)
        print(count)
    print("---")
    ys: list[int] = [1, 2, 3, 4, 5, 6, 7, 8]
    for k, group in groupby(ys, key=lambda x: x % 2):
        print(k)
        for v in group:
            print(v)


if __name__ == "__main__":
    main()
