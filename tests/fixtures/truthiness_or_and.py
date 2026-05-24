def main() -> None:
    empty: str = ""
    name: str = "world"
    print(empty or name)
    print(name or empty)
    print(empty or "fallback")
    print(name and "second")
    print(empty and "skipped")
    # list truthiness in if
    xs: list[int] = []
    ys: list[int] = [1, 2, 3]
    if xs:
        print("xs truthy")
    else:
        print("xs falsy")
    if ys:
        print("ys truthy")
    # negation
    if not xs:
        print("not xs")
    if not name:
        print("never")
    # ints
    n: int = 0
    m: int = 5
    if n:
        print("n true")
    else:
        print("n false")
    if m:
        print("m true")
    # dict
    d1: dict[str, int] = {}
    d2: dict[str, int] = {"a": 1}
    if d1:
        print("d1 true")
    else:
        print("d1 false")
    if d2:
        print("d2 true")


if __name__ == "__main__":
    main()
