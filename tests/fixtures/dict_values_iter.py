def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2}
    # check missing
    if "a" in d:
        print("a present")
    if "x" not in d:
        print("x absent")
    # iteration over keys
    keys: list[str] = sorted(d.keys())
    for k in keys:
        print(k)
    # iteration over values
    total: int = 0
    for v in d.values():
        total += v
    print(total)


if __name__ == "__main__":
    main()
