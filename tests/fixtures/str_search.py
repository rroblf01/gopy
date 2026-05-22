def main() -> None:
    s = "the quick brown fox jumps over the lazy dog"
    print(s.count("the"))
    print(s.find("fox"))
    print(s.find("cat"))
    print(s.rfind("the"))
    print(s.index("over"))
    parts = s.split()
    print(len(parts))
    print(parts[3])
    s2 = " trim me   "
    print(repr(s2.strip()))


if __name__ == "__main__":
    main()
