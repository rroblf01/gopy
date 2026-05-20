from datetime import datetime, date


def main() -> None:
    dt = datetime.fromisoformat("2026-05-19T12:34:56")
    n = dt.replace(year=2030, hour=0)
    print(n.year)
    print(n.month)
    print(n.day)
    print(n.hour)
    print(n.minute)
    d = date.fromisoformat("2026-05-19")
    e = d.replace(month=12, day=31)
    print(e.year)
    print(e.month)
    print(e.day)


if __name__ == "__main__":
    main()
