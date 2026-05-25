from datetime import datetime


def main() -> None:
    # Trailing Z (UTC marker)
    d1 = datetime.fromisoformat("2024-06-15T10:30:00Z")
    print(d1.strftime("%Y-%m-%dT%H:%M:%S"))

    # +HHMM offset
    d2 = datetime.fromisoformat("2024-01-15T10:30:00+0500")
    print(d2.strftime("%Y-%m-%dT%H:%M:%S"))

    # +HH:MM offset
    d3 = datetime.fromisoformat("2024-01-15T10:30:00+05:00")
    print(d3.strftime("%Y-%m-%dT%H:%M:%S"))

    # Negative offset
    d4 = datetime.fromisoformat("2024-12-31T23:59:00-08:00")
    print(d4.strftime("%Y-%m-%dT%H:%M:%S"))

    # Naive (no offset) — still works
    d5 = datetime.fromisoformat("2024-03-04T05:06:07")
    print(d5.strftime("%Y-%m-%dT%H:%M:%S"))


if __name__ == "__main__":
    main()
