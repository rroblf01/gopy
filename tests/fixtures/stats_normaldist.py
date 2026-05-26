import statistics


def main() -> None:
    n = statistics.NormalDist(0.0, 1.0)
    p = n.cdf(0.0)
    # Round to 4 places to avoid float drift.
    print(round(p, 4))
    print(n.mean)
    print(n.stdev)


if __name__ == "__main__":
    main()
