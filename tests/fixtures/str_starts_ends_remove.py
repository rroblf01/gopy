def main() -> None:
    print("hello".startswith("hel"))
    print("hello".startswith("foo"))
    print("hello".startswith(("hel", "foo")))
    print("hello".startswith(("bar", "foo")))
    print("file.txt".endswith(".txt"))
    print("file.txt".endswith((".log", ".txt")))
    print("file.txt".endswith((".log", ".md")))
    print("prefix_value".removeprefix("prefix_"))
    print("prefix_value".removeprefix("none_"))
    print("value_suffix".removesuffix("_suffix"))
    print("value_suffix".removesuffix("_none"))


if __name__ == "__main__":
    main()
