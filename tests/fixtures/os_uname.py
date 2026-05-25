import os


def main() -> None:
    u = os.uname()
    print(len(u))


if __name__ == "__main__":
    main()
