import sysconfig


def main() -> None:
    paths: dict[str, str] = sysconfig.get_paths()
    print("stdlib" in paths)
    print("purelib" in paths)
    print(len(sysconfig.get_platform()) > 0)
    print(len(sysconfig.get_python_version()) > 0)


if __name__ == "__main__":
    main()
