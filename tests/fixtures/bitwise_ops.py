def main() -> None:
    print(0b1100 | 0b1010)
    print(0b1100 & 0b1010)
    print(0b1100 ^ 0b1010)
    print(1 << 5)
    print(64 >> 2)
    print(~0)
    a: int = 0xff
    b: int = 0x0f
    print(a | b)
    print(a & b)
    print(a ^ b)
    print(a << 4)
    print(a >> 4)


if __name__ == "__main__":
    main()
