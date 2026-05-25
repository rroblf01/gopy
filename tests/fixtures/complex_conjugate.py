def main() -> None:
    c: complex = complex(2.0, 3.0)
    print(c.real)
    print(c.imag)
    cc: complex = c.conjugate()
    print(cc.real)
    print(cc.imag)


if __name__ == "__main__":
    main()
