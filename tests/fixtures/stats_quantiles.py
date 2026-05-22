import statistics


def main() -> None:
    xs: list[float] = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
    q = statistics.quantiles(xs)
    print(len(q))
    coeffs: list[float] = statistics.linear_regression([1.0, 2.0, 3.0, 4.0], [2.0, 4.0, 6.0, 8.0])
    print(round(coeffs[0], 4))
    print(round(coeffs[1], 4))


if __name__ == "__main__":
    main()
