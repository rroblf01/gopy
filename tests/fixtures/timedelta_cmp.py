from datetime import timedelta


def main() -> None:
    short = timedelta(seconds=30)
    long = timedelta(minutes=5)
    same = timedelta(seconds=30)

    print(short < long)
    print(short > long)
    print(short <= same)
    print(short >= same)
    print(short == same)
    print(short != long)


if __name__ == "__main__":
    main()
