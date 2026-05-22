import shutil


def main() -> None:
    total, used, free = shutil.disk_usage("/")
    print(total > 0)
    print(used >= 0)
    print(free >= 0)
    cols, rows = shutil.get_terminal_size()
    print(cols > 0)
    print(rows > 0)


if __name__ == "__main__":
    main()
