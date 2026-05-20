import re


def main() -> None:
    p = re.compile(r"\s+")
    for v in p.split("hello   world  foo"):
        print(v)
    print("---")
    res = p.subn("_", "a b  c")
    print(res[0])
    print(res[1])
    q = re.compile(r"\d+")
    m = q.fullmatch("123")
    if m is not None:
        print(m.group(0))
    n = q.fullmatch("12a")
    if n is None:
        print("no")


if __name__ == "__main__":
    main()
