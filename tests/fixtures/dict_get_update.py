def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2}
    print(d.get("a", -1))
    print(d.get("x", -1))
    print(d.get("a"))
    d.update({"c": 3, "d": 4})
    for k in sorted(d.keys()):
        print(k, d[k])


if __name__ == "__main__":
    main()
