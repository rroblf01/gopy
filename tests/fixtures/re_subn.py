import re


def main() -> None:
    res = re.subn(r"\d+", "N", "a1 b22 c333")
    print(res[0])
    print(res[1])
    res2 = re.subn(r"xyz", "?", "abc def")
    print(res2[0])
    print(res2[1])


if __name__ == "__main__":
    main()
