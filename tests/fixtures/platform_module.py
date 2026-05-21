import platform


def main() -> None:
    sysname: str = platform.system()
    print(sysname in ["Linux", "Darwin", "Windows", "FreeBSD", "OpenBSD", "NetBSD"])
    arch: str = platform.machine()
    print(len(arch) > 0)
    ver: str = platform.python_version()
    print(len(ver) > 0)
    plat: str = platform.platform()
    print("-" in plat)


if __name__ == "__main__":
    main()
