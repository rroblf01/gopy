import time


def main() -> None:
    tup = time.gmtime(0)
    print(tup[0])
    print(tup[1])
    print(tup[2])
    s: str = time.strftime("%Y-%m-%d", tup)
    print(s)


if __name__ == "__main__":
    main()
