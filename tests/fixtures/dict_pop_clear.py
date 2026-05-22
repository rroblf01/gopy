def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2, "c": 3}
    print(d.pop("a"))
    print(d.pop("x", -1))
    print(sorted(d.keys()))
    d["d"] = 4
    d.clear()
    print(len(d))


if __name__ == "__main__":
    main()
