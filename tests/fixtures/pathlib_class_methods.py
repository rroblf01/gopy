from pathlib import Path


def main() -> None:
    p = Path("foo/bar/baz.txt")
    print(p.match("*.txt"))
    print(p.match("baz.txt"))
    print(p.match("*.py"))
    # Multi-segment pattern matches the joined path.
    print(p.match("foo/bar/*.txt"))


if __name__ == "__main__":
    main()
