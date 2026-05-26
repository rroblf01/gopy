import statistics


def main() -> None:
    xs: list[int] = [1, 2, 2, 3, 3, 4]
    modes: list[int] = statistics.multimode(xs)
    modes.sort()
    print(len(modes))
    for m in modes:
        print(m)

    ys: list[int] = [7, 7, 7]
    only: list[int] = statistics.multimode(ys)
    print(len(only))
    print(only[0])


if __name__ == "__main__":
    main()
