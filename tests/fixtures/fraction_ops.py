from fractions import Fraction


def main() -> None:
    a = Fraction(1, 2)
    b = Fraction(1, 3)
    print(a + b)
    print(a - b)
    print(a * b)
    print(a / b)
    print(a.numerator, a.denominator)
    print(a == b)
    print(a != b)
    print(a < b)
    print(a > b)
    print(a <= Fraction(1, 2))
    print(a >= Fraction(1, 2))


if __name__ == "__main__":
    main()
