import struct


def main() -> None:
    s = struct.Struct("<I")
    print(s.calcsize())
    packed = s.pack(42)
    parts = s.unpack(packed)
    print(parts[0])


if __name__ == "__main__":
    main()
