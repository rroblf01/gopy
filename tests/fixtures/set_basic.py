def main() -> None:
    xs: list[int] = [1, 2, 3, 2, 1, 4, 3]
    uniq = set(xs)
    for v in sorted(uniq):
        print(v)
    print(len(uniq))
    fz = frozenset([5, 5, 6, 6, 7])
    for v in sorted(fz):
        print(v)
    print(3 in uniq)
    print(99 in uniq)


if __name__ == "__main__":
    main()
