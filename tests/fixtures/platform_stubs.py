import platform


def main() -> None:
    mv = platform.mac_ver()
    print(len(mv))
    wv = platform.win32_ver()
    print(len(wv))


if __name__ == "__main__":
    main()
