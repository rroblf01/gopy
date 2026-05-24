def main() -> None:
    a: list[int] = [1, 2, 3]
    b: list[int] = [1, 2, 3]
    c: list[int] = [1, 2, 4]
    print(a == b)
    print(a == c)
    print(a != c)
    print(a != b)
    d1: dict[str, int] = {"a": 1, "b": 2}
    d2: dict[str, int] = {"a": 1, "b": 2}
    d3: dict[str, int] = {"a": 1, "b": 3}
    print(d1 == d2)
    print(d1 == d3)
    nested1: list[list[int]] = [[1, 2], [3, 4]]
    nested2: list[list[int]] = [[1, 2], [3, 4]]
    nested3: list[list[int]] = [[1, 2], [3, 5]]
    print(nested1 == nested2)
    print(nested1 == nested3)
    # in conditional
    if a == b:
        print("eq")
    if a != c and len(a) == 3:
        print("diff")


if __name__ == "__main__":
    main()
