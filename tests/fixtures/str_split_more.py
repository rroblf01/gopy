def main() -> None:
    parts = "a,b,c,d,e".split(",", 2)
    for p in parts:
        print(p)
    print("---")
    lines = "hello\nworld\nthird".splitlines()
    for line in lines:
        print(line)
    print("---")
    pre = "key=value=extra".partition("=")
    for x in pre:
        print(x)
    print("---")
    rpre = "key=value=extra".rpartition("=")
    for x in rpre:
        print(x)


if __name__ == "__main__":
    main()
