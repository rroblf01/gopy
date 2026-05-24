def main() -> None:
    n: int = 258
    bb: bytes = n.to_bytes(4, "big")
    bl: bytes = n.to_bytes(4, "little")
    print(len(bb))
    print(len(bl))
    print(int.from_bytes(bb, "big"))
    print(int.from_bytes(bl, "little"))


if __name__ == "__main__":
    main()
