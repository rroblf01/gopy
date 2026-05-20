import calendar


def main() -> None:
    print(calendar.isleap(2024))
    print(calendar.isleap(2023))
    print(calendar.isleap(2000))
    print(calendar.isleap(1900))
    pair = calendar.monthrange(2026, 5)
    print(pair[0])
    print(pair[1])
    print(calendar.month_name[1])
    print(calendar.month_name[5])
    print(calendar.day_name[0])
    print(calendar.day_name[6])
    print(calendar.weekday(2026, 5, 19))


if __name__ == "__main__":
    main()
