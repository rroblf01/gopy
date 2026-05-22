def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2, "c": 3}
    keys = sorted(d.keys())
    print(keys)
    vals = sorted(d.values())
    print(vals)
    # has-key
    print("a" in d)
    print("x" in d)
    d["d"] = 4
    print(len(d))


if __name__ == "__main__":
    main()
