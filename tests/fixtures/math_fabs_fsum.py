import math


def main() -> None:
    print(math.fabs(-3.5))
    print(math.ldexp(1.5, 3))
    xs: list[float] = [0.1, 0.2, 0.3, 0.4]
    print(round(math.fsum(xs), 6))
    print(math.ulp(1.0) > 0)


if __name__ == "__main__":
    main()
