from datetime import date, datetime


def main() -> None:
    d = date(2026, 5, 19)
    print(d.strftime("%Y-%m-%d"))
    print(d.strftime("%d/%m/%Y"))
    print(d.strftime("%B %d, %Y"))
    dt = datetime.strptime("2026-05-19 12:34:56", "%Y-%m-%d %H:%M:%S")
    print(dt.strftime("%Y-%m-%d %H:%M:%S"))
    print(dt.year)
    print(dt.month)
    print(dt.day)
    print(dt.hour)
    print(dt.minute)
    print(dt.second)


if __name__ == "__main__":
    main()
