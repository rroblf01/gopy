from datetime import date, time


def main() -> None:
    d = date(2026, 5, 19)
    print(d.isoformat())
    print(d.year)
    print(d.month)
    print(d.day)
    t = time(12, 34, 56)
    print(t.isoformat())
    print(t.hour)
    print(t.minute)
    print(t.second)


if __name__ == "__main__":
    main()
