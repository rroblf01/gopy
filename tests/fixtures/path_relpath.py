import os.path


def main() -> None:
    print(os.path.relpath("/a/b/c", "/a"))
    print(os.path.relpath("/a/b/c", "/a/b"))
    print(os.path.relpath("/a/b/c/d", "/a/b/x"))


if __name__ == "__main__":
    main()
