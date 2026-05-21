def main() -> None:
    a: complex = complex(1, 2)
    b: complex = complex(3, 4)
    print(a.real)
    print(a.imag)
    c: complex = a + b
    print(c.real)
    print(c.imag)
    d: complex = a * b
    print(d.real)
    print(d.imag)
    e: complex = complex(0, 5)
    print(e)
    f: complex = complex(2, -3)
    print(f)


if __name__ == "__main__":
    main()
