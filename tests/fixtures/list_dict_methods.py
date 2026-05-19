def main() -> None:
    xs: list[int] = [1, 2, 3, 2, 1, 2]
    print(xs.count(2))
    print(xs.count(99))
    print(xs.index(3))
    print(xs.index(2))
    d: dict[str, int] = {"a": 1, "b": 2}
    e: dict[str, int] = {"b": 99, "c": 3}
    d.update(e)
    print(d["a"])
    print(d["b"])
    print(d["c"])
    print(len(d))


if __name__ == "__main__":
    main()
