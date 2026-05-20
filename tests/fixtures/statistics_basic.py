import statistics


def main() -> None:
    xs: list[float] = [1.0, 2.0, 3.0, 4.0, 5.0]
    print(statistics.mean(xs))
    print(statistics.median(xs))
    print(round(statistics.variance(xs), 4))
    print(round(statistics.stdev(xs), 4))
    print(round(statistics.pstdev(xs), 4))
    ys: list[int] = [1, 2, 2, 3, 3, 3, 4]
    print(statistics.mode(ys))


if __name__ == "__main__":
    main()
