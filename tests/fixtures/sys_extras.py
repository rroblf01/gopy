import sys


def main() -> None:
    print(sys.maxunicode)
    print(sys.getrecursionlimit())
    print(sys.getdefaultencoding())
    print(sys.getfilesystemencoding())
    print(sys.dont_write_bytecode)
    print(len(sys.builtin_module_names) > 0)


if __name__ == "__main__":
    main()
