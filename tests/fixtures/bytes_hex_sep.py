def main() -> None:
    s: str = "Hello"
    print(s.encode().hex())
    print(s.encode().hex(":"))
    # bytearray.fromhex round-trip with separator
    raw: str = bytes.fromhex("ff 00 7f")
    print(raw.hex(" "))


if __name__ == "__main__":
    main()
