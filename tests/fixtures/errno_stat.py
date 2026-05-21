import errno
import stat


def main() -> None:
    print(errno.ENOENT == 2)
    print(errno.EACCES == 13)
    print(stat.S_IRUSR == 0o400)
    print(stat.S_IWUSR == 0o200)
    print(stat.S_ISDIR(stat.S_IFDIR | stat.S_IRUSR))
    print(stat.S_ISREG(stat.S_IFREG))
    print(stat.S_ISDIR(stat.S_IFREG))


if __name__ == "__main__":
    main()
