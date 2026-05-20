import bisect


def main() -> None:
    xs: list[int] = [1, 3, 5, 7, 9]
    print(bisect.bisect_left(xs, 4))
    print(bisect.bisect_right(xs, 5))
    print(bisect.bisect_left(xs, 5))
    print(bisect.bisect_right(xs, 0))
    print(bisect.bisect_left(xs, 100))
    bisect.insort(xs, 4)
    for v in xs:
        print(v)
    bisect.insort(xs, 0)
    print(xs[0])
    print(xs[len(xs) - 1])


if __name__ == "__main__":
    main()
