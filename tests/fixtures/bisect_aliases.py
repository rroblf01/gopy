import bisect


def main() -> None:
    xs: list[int] = [1, 3, 5, 7, 9]
    print(bisect.bisect(xs, 4))
    print(bisect.bisect(xs, 5))
    print(bisect.bisect_left(xs, 5))
    print(bisect.bisect_right(xs, 5))
    ys: list[int] = [1, 3, 5]
    bisect.insort_left(ys, 4)
    print(len(ys))
    print(ys[2])
    zs: list[int] = [1, 3, 5]
    bisect.insort_right(zs, 4)
    print(len(zs))
    print(zs[2])


if __name__ == "__main__":
    main()
