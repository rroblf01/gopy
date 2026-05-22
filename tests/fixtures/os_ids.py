import os


def main() -> None:
    print(os.getpid() > 0)
    print(os.getppid() > 0)
    print(os.getuid() >= 0)
    print(os.geteuid() >= 0)
    print(os.getgid() >= 0)
    print(os.fspath("hello"))


if __name__ == "__main__":
    main()
