def main() -> None:
    a = 0b1010
    b = 0b1100
    print(a & b)
    print(a | b)
    print(a ^ b)
    print(~a)
    print(a << 2)
    print(b >> 1)
    print(hex(255))
    print(oct(8))
    print(bin(10))
    print(int("0xff", 16))
    print(int("0o17", 8))
    print(int("0b101", 2))
    print(int("ff", 16))
    print(int("101", 2))
    print(int("-1a", 16))


if __name__ == "__main__":
    main()
