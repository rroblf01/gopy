def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2, "c": 3}
    del d["b"]
    print(len(d))
    print("a" in d)
    print("b" in d)

    xs: list[int] = [10, 20, 30, 40]
    del xs[1]
    print(len(xs))
    print(xs[0])
    print(xs[1])
    print(xs[2])


if __name__ == "__main__":
    main()
