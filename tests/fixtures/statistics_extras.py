import statistics


def main() -> None:
    xs: list[float] = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0]
    print(statistics.median_low(xs))
    print(statistics.median_high(xs))
    odd: list[float] = [1.0, 2.0, 3.0, 4.0, 5.0]
    print(statistics.median_low(odd))
    print(statistics.median_high(odd))
    print(round(statistics.harmonic_mean([2.0, 4.0, 4.0]), 4))
    print(round(statistics.pvariance([1.0, 2.0, 3.0, 4.0, 5.0]), 4))


if __name__ == "__main__":
    main()
