import stat


def main() -> None:
    print(stat.filemode(0o100644))
    print(stat.filemode(0o040755))
    print(stat.filemode(0o120777))
    print(stat.filemode(0o100000))
    print(stat.filemode(0o100755 | stat.S_ISUID))


if __name__ == "__main__":
    main()
