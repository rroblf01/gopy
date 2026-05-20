import re


def main() -> None:
    for part in re.split(r"\s+", "hello  world  foo"):
        print(part)
    print("---")
    print(re.escape("a.b*c?"))
    print("---")
    m = re.fullmatch(r"\d+", "12345")
    if m is not None:
        print(m.group(0))
    n = re.fullmatch(r"\d+", "12a")
    if n is None:
        print("no match")


if __name__ == "__main__":
    main()
