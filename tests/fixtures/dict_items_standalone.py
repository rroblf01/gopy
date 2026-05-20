def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2, "c": 3}
    pairs = d.items()
    print(len(pairs))
    # Iterate via tuple-unpack (existing path) to verify content
    total: int = 0
    for _, v in d.items():
        total = total + v
    print(total)


if __name__ == "__main__":
    main()
