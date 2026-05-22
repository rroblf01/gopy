import os


def main() -> None:
    print(os.curdir)
    print(os.pardir)
    print(os.extsep)
    print(len(os.devnull) > 0)
    print(os.pathsep in (":", ";"))


if __name__ == "__main__":
    main()
