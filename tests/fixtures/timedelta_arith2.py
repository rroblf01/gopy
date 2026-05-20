from datetime import timedelta


def main() -> None:
    a = timedelta(hours=1)
    b = a * 3
    print(b)
    c = 2 * a
    print(c)
    d = b - a
    print(d)
    e = b / 2
    print(e)
    f = a + b
    print(f)


if __name__ == "__main__":
    main()
