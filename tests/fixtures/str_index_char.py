def first_char(s: str) -> str:
    return s[0]


def main() -> None:
    s = "hello"
    print(s[0])
    print(s[1])
    print(s[-1])
    print(s[-2])
    # in conditional
    if s[0] == "h":
        print("starts h")
    # iterate index
    for i in range(len(s)):
        print(s[i])
    # accumulate first chars
    words = ["apple", "ant", "banana", "berry", "cherry"]
    firsts: list[str] = []
    for w in words:
        firsts.append(w[0])
    print(firsts)
    # str arg passed back
    print(first_char("zebra"))


if __name__ == "__main__":
    main()
