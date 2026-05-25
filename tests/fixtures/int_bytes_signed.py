def main() -> None:
    raw: str = (-2).to_bytes(2, "big", signed=True)
    print(len(raw))
    back: int = int.from_bytes(raw, "big", signed=True)
    print(back)
    raw2: str = (250).to_bytes(1, "little", signed=False)
    print(int.from_bytes(raw2, "little"))
    raw3: str = (-128).to_bytes(1, "big", signed=True)
    print(int.from_bytes(raw3, "big", signed=True))


if __name__ == "__main__":
    main()
