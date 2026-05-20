from datetime import date


def main() -> None:
    d = date.fromisoformat("2026-05-19")
    print(d.year)
    print(d.month)
    print(d.day)
    print(d.isoformat())
    print(d.weekday())
    print(d.isoweekday())
    t = date.today()
    print(t.year > 2000)


if __name__ == "__main__":
    main()
