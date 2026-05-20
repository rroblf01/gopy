from datetime import datetime


def main() -> None:
    d = datetime.fromisoformat("2026-05-19T12:34:56")
    print(d.year)
    print(d.month)
    print(d.day)
    print(d.hour)
    print(d.minute)
    print(d.second)
    print(d.strftime("%Y-%m-%d"))
    e = datetime.fromisoformat("2020-01-15")
    print(e.year)
    print(e.month)
    print(e.day)


if __name__ == "__main__":
    main()
