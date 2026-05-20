from datetime import timedelta


def main() -> None:
    print(timedelta(days=1))
    print(timedelta(hours=5))
    print(timedelta(hours=1, minutes=30))
    print(timedelta(days=2, hours=3, minutes=4, seconds=5))
    print(timedelta(weeks=1))
    print(timedelta(minutes=90))
    print(timedelta(0))
    print(timedelta(days=0, seconds=3600))


if __name__ == "__main__":
    main()
