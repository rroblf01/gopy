import os


def main() -> None:
    st = os.statvfs("/")
    print(len(st))


if __name__ == "__main__":
    main()
