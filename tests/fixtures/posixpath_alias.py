import posixpath


def main() -> None:
    print(posixpath.join("a", "b", "c"))
    print(posixpath.basename("/foo/bar/baz.txt"))
    print(posixpath.dirname("/foo/bar/baz.txt"))


if __name__ == "__main__":
    main()
