import ntpath


def main() -> None:
    print(ntpath.basename("/foo/bar/baz.txt"))
    print(ntpath.dirname("/foo/bar/baz.txt"))


if __name__ == "__main__":
    main()
