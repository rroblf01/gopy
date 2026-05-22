import calendar


def main() -> None:
    weeks = calendar.monthcalendar(2024, 2)
    print(len(weeks))
    for w in weeks:
        for d in w:
            print(d)
    print(calendar.leapdays(2000, 2024))


if __name__ == "__main__":
    main()
