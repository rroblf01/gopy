import re


def main() -> None:
    p = re.compile(r"(\w+)=(\d+)")
    m = p.search("x=42 y=7")
    if m is not None:
        print(m.group(0))
        print(m.group(1))
        print(m.group(2))
    q = re.compile(r"\w+=\d+")
    for hit in q.findall("a=1 b=22 c=333"):
        print(hit)
    print(p.sub("X=Y", "x=42 y=7"))
    p2 = re.compile(r"\d+")
    print(p2.match("123abc") is not None)
    print(p2.match("abc123") is not None)


if __name__ == "__main__":
    main()
