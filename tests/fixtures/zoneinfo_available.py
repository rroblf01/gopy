import zoneinfo


def main() -> None:
    zones = zoneinfo.available_timezones()
    # CPython returns a set, gopy a list — compare counts via length.
    print(len(zones) > 0)


if __name__ == "__main__":
    main()
