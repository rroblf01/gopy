import math


def main() -> None:
    print(math.isclose(1.0, 1.0))
    print(math.isclose(1.0, 1.0000000001))
    print(math.isclose(1.0, 2.0))
    print(round(math.asin(1.0), 4))
    print(round(math.acos(0.0), 4))
    print(round(math.sinh(0.0), 4))
    print(round(math.cosh(0.0), 4))
    print(round(math.tanh(0.0), 4))
    print(round(math.expm1(0.0), 4))
    print(round(math.log1p(0.0), 4))
    print(round(math.atanh(0.5), 4))


if __name__ == "__main__":
    main()
