import re


def main() -> None:
    text: str = "name=ada age=36"
    m = re.search("name=([a-z]+)", text)
    if m is not None:
        print(m.group())
        print(m.group(1))
    miss = re.search("zzz", text)
    if miss is None:
        print("no match")
    if miss:
        print("nope")
    else:
        print("falsy")
    head = re.match("name", text)
    if head:
        print(head.group())


if __name__ == "__main__":
    main()
