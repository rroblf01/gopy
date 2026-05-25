def main() -> None:
    raw: str = bytes.fromhex("48656c6c6f")
    print(len(raw))
    spaced: str = bytes.fromhex("ff 00 7f")
    print(len(spaced))
    # hex() round-trip avoids printing the raw bytes (which CPython
    # renders as b'...' but gopy treats as str).
    print(spaced.hex())


if __name__ == "__main__":
    main()
