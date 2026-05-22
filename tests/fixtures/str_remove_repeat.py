def main() -> None:
    s = "abc-def-ghi"
    print(s.startswith("abc"))
    print(s.startswith("xyz"))
    print(s.endswith("ghi"))
    print(s.removeprefix("abc-"))
    print(s.removesuffix("-ghi"))
    print(s * 3)
    print("=" * 5)
    s2 = "  spaces  "
    print(repr(s2.strip()))
    print(repr(s2.lstrip()))
    print(repr(s2.rstrip()))


if __name__ == "__main__":
    main()
