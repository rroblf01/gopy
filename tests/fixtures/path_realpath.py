import os


def main() -> None:
    abs1: str = os.path.realpath("/tmp")
    print(len(abs1) > 0)
    print(os.path.isabs(abs1))
    common: str = os.path.commonpath(["/a/b/c", "/a/b/d"])
    print(common == "/a/b")
    common2: str = os.path.commonpath(["/x/y", "/z"])
    print(common2 == "/" or common2 == "")


if __name__ == "__main__":
    main()
