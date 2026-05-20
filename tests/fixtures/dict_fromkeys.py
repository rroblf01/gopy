def main() -> None:
    d: dict[str, int] = dict.fromkeys(["a", "b", "c"], 0)
    for k in sorted(d.keys()):
        print(k)
        print(d[k])
    d["b"] = 99
    print(d["b"])
    e: dict[int, str] = dict.fromkeys([1, 2, 3], "x")
    for k in sorted(e.keys()):
        print(k)
        print(e[k])


if __name__ == "__main__":
    main()
