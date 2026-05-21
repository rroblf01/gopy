import sys


def main() -> None:
    print(sys.platform in ["linux", "darwin", "windows", "freebsd", "openbsd", "netbsd"])
    print(sys.byteorder in ["little", "big"])
    print(sys.maxsize > 0)
    print(len(sys.version) > 0)


if __name__ == "__main__":
    main()
