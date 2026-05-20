def main() -> None:
    xs: list[int] = [1, 2, 3]
    ys = xs.copy()
    ys.append(99)
    for v in xs:
        print(v)
    print("---")
    for v in ys:
        print(v)
    print("---")
    d: dict[str, int] = {"a": 1, "b": 2}
    e = d.copy()
    e["c"] = 3
    print(d.get("c", -1))
    print(e["c"])
    print(len(d))
    print(len(e))


if __name__ == "__main__":
    main()
