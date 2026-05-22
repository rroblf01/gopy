import shutil


def main() -> None:
    sz: list[int] = shutil.disk_usage("/")
    print(len(sz))
    print(sz[0] > 0)
    ts: list[int] = shutil.get_terminal_size()
    print(ts[0] > 0)
    print(ts[1] > 0)


if __name__ == "__main__":
    main()
