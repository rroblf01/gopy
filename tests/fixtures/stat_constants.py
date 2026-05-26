import stat


def main() -> None:
    print(stat.ST_MODE)
    print(stat.ST_SIZE)
    print(stat.S_IFREG)
    print(stat.S_IFDIR)


if __name__ == "__main__":
    main()
