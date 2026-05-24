import re


def main() -> None:
    m = re.search("hello", "HELLO world", re.IGNORECASE)
    print(m.group() if m else "None")

    matches = re.findall("^line", "line1\nline2\nline3", re.MULTILINE)
    print(matches)

    m2 = re.search("a.b", "a\nb", re.DOTALL)
    print(m2.group() if m2 else "None")

    m3 = re.search("hello", "HELLO world")
    print(m3 is None)

    p = re.compile("foo", re.IGNORECASE)
    print(p.findall("Foo FOO foo bar"))

    p2 = re.compile("HELLO", re.IGNORECASE)
    print(p2.sub("hi", "Hello world"))


if __name__ == "__main__":
    main()
