import os


def main() -> None:
    g = os.getgroups()
    print(len(g) >= 0)


if __name__ == "__main__":
    main()
