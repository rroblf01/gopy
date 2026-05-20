def main() -> None:
    nested = {x * 10 + y: x * y for x in [1, 2, 3] for y in [4, 5] if x != 2}
    vals: list[int] = []
    for k in nested:
        vals.append(nested[k])
    vals.sort()
    for v in vals:
        print(v)
    print("---")
    d: dict[str, int] = {"a": 1}
    pair = d.popitem()
    print(pair[0])
    print(pair[1])
    print(len(d))


if __name__ == "__main__":
    main()
