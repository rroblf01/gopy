import cmath


def main() -> None:
    print(cmath.sqrt(complex(-1, 0)))
    c: complex = complex(3, 4)
    print(abs(c))
    print(round(cmath.phase(complex(0, 1)), 4))
    print(round(cmath.phase(complex(1, 0)), 4))
    pair = cmath.polar(complex(1, 0))
    print(pair[0])
    print(pair[1])
    print(cmath.rect(1.0, 0.0))
    print(cmath.isnan(complex(1, 0)))
    print(cmath.isinf(complex(1, 0)))
    print(round(cmath.pi, 4))
    print(round(cmath.tau, 4))


if __name__ == "__main__":
    main()
