def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2, "c": 3}
    total: int = 0
    for k, v in d.items():
        total += v
    print(total)
    # iterate with sorted keys
    out: list[str] = []
    for k, v in d.items():
        out.append(f"{k}={v}")
    out.sort()
    print(out)


if __name__ == "__main__":
    main()
