import struct


def main() -> None:
    print(struct.calcsize("<I"))
    print(struct.calcsize(">Q"))
    packed = struct.pack("<I", 42)
    out = struct.unpack("<I", packed)
    print(out[0])


if __name__ == "__main__":
    main()
