def main() -> None:
    s: str = "hello world"
    print("hello" in s)
    print("xyz" in s)
    print("xyz" not in s)

    xs: list[int] = [1, 2, 3, 4]
    print(2 in xs)
    print(99 in xs)
    print(99 not in xs)

    d: dict[str, int] = {"a": 1, "b": 2}
    print("a" in d)
    print("z" in d)
    print("z" not in d)

    # Set literal lowers to the same slice shape; in / not in still work.
    members = {10, 20, 30}
    print(20 in members)
    print(40 in members)


if __name__ == "__main__":
    main()
