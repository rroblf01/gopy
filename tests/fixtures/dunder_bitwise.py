class Bits:
    def __init__(self, v: int) -> None:
        self.v = v

    def __or__(self, other: "Bits") -> "Bits":
        return Bits(self.v | other.v)

    def __and__(self, other: "Bits") -> "Bits":
        return Bits(self.v & other.v)

    def __xor__(self, other: "Bits") -> "Bits":
        return Bits(self.v ^ other.v)

    def __lshift__(self, k: int) -> "Bits":
        return Bits(self.v << k)

    def __rshift__(self, k: int) -> "Bits":
        return Bits(self.v >> k)

    def __invert__(self) -> "Bits":
        return Bits(~self.v)


def main() -> None:
    a = Bits(0b1100)
    b = Bits(0b1010)
    print((a | b).v)
    print((a & b).v)
    print((a ^ b).v)
    print((a << 2).v)
    print((a >> 1).v)
    print((~a).v)


if __name__ == "__main__":
    main()
