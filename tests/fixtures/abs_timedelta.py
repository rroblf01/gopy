from datetime import timedelta


def main() -> None:
    pos = timedelta(hours=5)
    print(abs(pos))
    neg = timedelta(hours=-3)
    print(abs(neg))


if __name__ == "__main__":
    main()
