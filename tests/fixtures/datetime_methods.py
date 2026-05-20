from datetime import datetime, timedelta


def main() -> None:
    d = datetime.fromisoformat("2026-05-19T12:34:56")
    # 2026-05-19 is a Tuesday
    print(d.weekday())
    print(d.isoweekday())
    print(d.timestamp() > 0)
    td = timedelta(days=2, hours=3)
    print(td.total_seconds())
    print(td.days)
    print(td.seconds)


if __name__ == "__main__":
    main()
