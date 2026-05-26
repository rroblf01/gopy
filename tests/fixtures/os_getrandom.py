import os


def main() -> None:
    b = os.getrandom(16)
    print(len(b))


if __name__ == "__main__":
    main()
