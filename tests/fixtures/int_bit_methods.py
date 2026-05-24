def main() -> None:
    a: int = 5
    b: int = 255
    c: int = 0
    d: int = -3
    print(a.bit_length())
    print(b.bit_length())
    print(c.bit_length())
    print(d.bit_length())
    print(a.bit_count())
    print(b.bit_count())
    print(c.bit_count())
    print(d.bit_count())


if __name__ == "__main__":
    main()
