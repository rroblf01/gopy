def main() -> None:
    d = {"a": 1, "b": 2}
    print(d.pop("a"))
    print(sorted(d.keys()))
    try:
        v = d.pop("z")
        print(v)
    except KeyError:
        print("missing")
    # pop with default doesn't raise
    print(d.pop("z", -1))
    print(d.pop("b", 99))
    print(d.pop("b", 99))  # gone now → default


if __name__ == "__main__":
    main()
