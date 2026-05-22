def main() -> None:
    d: dict[str, int] = {}
    d["a"] = 1
    d["b"] = 2
    d["c"] = 3
    for k in sorted(d.keys()):
        print(k, d[k])
    summed: int = 0
    for k in sorted(d.keys()):
        summed += d[k]
    print(summed)
    keys = list(sorted(d.keys()))
    print(keys)


if __name__ == "__main__":
    main()
