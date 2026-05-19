import re


def main() -> None:
    m = re.search(r"(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})", "log 2026-05-19 today")
    if m is not None:
        print(m.group("year"))
        print(m.group("month"))
        print(m.group("day"))
        print(m.group(1))
        print(m.group(0))
    p = re.compile(r"(?P<key>\w+)=(?P<val>\d+)")
    n = p.search("x=42")
    if n is not None:
        print(n.group("key"))
        print(n.group("val"))


if __name__ == "__main__":
    main()
