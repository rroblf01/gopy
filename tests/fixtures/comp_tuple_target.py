def main() -> None:
    pairs = [(1, "a"), (2, "b"), (3, "c")]
    keys = [k for (k, v) in pairs]
    print(keys)
    vals = [v for (k, v) in pairs]
    print(vals)
    d = {k: v for (k, v) in pairs}
    for k in [1, 2, 3]:
        print(k, d[k])


if __name__ == "__main__":
    main()
