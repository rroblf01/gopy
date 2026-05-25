def main() -> None:
    raw: str = bytearray.fromhex("48 65 6c 6c 6f")
    print(len(raw))
    print(raw.hex())


if __name__ == "__main__":
    main()
