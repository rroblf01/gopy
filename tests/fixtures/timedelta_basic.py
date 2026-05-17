from datetime import timedelta


def main() -> None:
    d1 = timedelta(1)
    print(str(d1))
    d2 = timedelta(7)
    print(str(d2))
    d3 = timedelta(0)
    print(str(d3))


if __name__ == "__main__":
    main()
