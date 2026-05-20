import re


def main() -> None:
    m = re.search(r"(\d+)-(\w+)", "abc 123-foo def")
    if m is not None:
        print(m.start())
        print(m.end())
        sp = m.span()
        print(sp[0])
        print(sp[1])
        print(m.start(1))
        print(m.end(1))
        print(m.start(2))
        print(m.end(2))
        print(m.group(0))
        print(m.group(1))
        print(m.group(2))


if __name__ == "__main__":
    main()
