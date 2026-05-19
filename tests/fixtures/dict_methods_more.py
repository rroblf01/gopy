def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2, "c": 3}
    print(d.pop("b"))
    print(len(d))
    print(d.pop("missing", -1))
    print(d.setdefault("a", 99))
    print(d.setdefault("z", 50))
    print(d["z"])
    print(len(d))
    d.clear()
    print(len(d))


if __name__ == "__main__":
    main()
