import os


def main() -> None:
    p = os.pipe()
    print(len(p))
    print(p[0] > 0)
    os.close(p[0])
    os.close(p[1])


if __name__ == "__main__":
    main()
