def main() -> None:
    xs: list[str] = ["a", "b", "c"]
    pairs = enumerate(xs)
    for p in pairs:
        print(p[0], p[1])
    starts = enumerate(xs, 10)
    for p in starts:
        print(p[0], p[1])

    ys: list[int] = [1, 2, 3]
    zs: list[str] = ["x", "y", "z"]
    zipped = zip(ys, zs)
    for p in zipped:
        print(p[0], p[1])

    short: list[int] = [1, 2]
    long: list[int] = [10, 20, 30, 40]
    z2 = zip(short, long)
    total: int = 0
    for p in z2:
        total = total + int(p[0])
    print(total)


if __name__ == "__main__":
    main()
