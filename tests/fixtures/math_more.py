import math


def main() -> None:
    print(math.trunc(3.7))
    print(math.trunc(-3.7))
    print(math.fmod(10.5, 3.0))
    print(math.gcd(12, 18))
    print(math.gcd(17, 5))
    print(math.isnan(float("nan")))
    print(math.isnan(1.0))
    print(math.isinf(math.inf))
    print(math.isinf(1.0))
    print(math.isfinite(1.0))
    print(math.isfinite(math.inf))
    print(math.copysign(3.0, -1.0))
    print(math.hypot(3.0, 4.0))
    print(round(math.degrees(math.pi), 4))
    print(round(math.radians(180.0), 4))


if __name__ == "__main__":
    main()
