import glob


def main() -> None:
    print(glob.has_magic("foo*.py"))
    print(glob.has_magic("foo.py"))
    print(glob.has_magic("[abc].txt"))
    print(glob.escape("file*.txt"))


if __name__ == "__main__":
    main()
