import stat


def main() -> None:
    print(stat.S_IFCHR == 0o020000)
    print(stat.S_IFBLK == 0o060000)
    print(stat.S_IFIFO == 0o010000)
    print(stat.S_IFSOCK == 0o140000)
    print(stat.S_ISCHR(stat.S_IFCHR))
    print(stat.S_ISBLK(stat.S_IFBLK))
    print(stat.S_ISFIFO(stat.S_IFIFO))
    print(stat.S_ISSOCK(stat.S_IFSOCK))
    print(stat.S_IRWXU == 0o700)
    print(stat.S_IMODE(0o100644) == 0o644)


if __name__ == "__main__":
    main()
