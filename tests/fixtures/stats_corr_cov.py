import statistics


def main() -> None:
    xs: list[float] = [1.0, 2.0, 3.0, 4.0, 5.0]
    ys: list[float] = [2.0, 4.0, 6.0, 8.0, 10.0]
    print(round(statistics.correlation(xs, ys), 4))
    print(round(statistics.covariance(xs, ys), 4))
    print(round(statistics.geometric_mean([1.0, 2.0, 4.0, 8.0]), 4))


if __name__ == "__main__":
    main()
