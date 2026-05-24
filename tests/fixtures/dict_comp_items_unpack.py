def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2, "c": 3}
    # invert dict via tuple-unpack target
    inv: dict[int, str] = {v: k for k, v in d.items()}
    for k in sorted(inv.keys()):
        print(k, inv[k])
    # list comp with tuple-unpack target
    pairs: list[str] = [f"{k}:{v}" for k, v in d.items()]
    pairs.sort()
    print(pairs)
    # filter via tuple-unpack
    big: dict[str, int] = {k: v for k, v in d.items() if v > 1}
    for k in sorted(big.keys()):
        print(k, big[k])


if __name__ == "__main__":
    main()
