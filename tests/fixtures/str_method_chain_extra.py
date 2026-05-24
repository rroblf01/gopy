def main() -> None:
    # chained str methods now infer through return types
    print("HELLO".casefold().upper())
    print("  Hi  ".strip().lower().title())
    print("Hello".swapcase().swapcase())
    print("a.b.c".replace(".", "-").upper())
    print("hello".center(11, "*").strip("*"))
    print("a,b,c".split(",")[1])
    # int returns
    print("abc".count("a") + "abc".count("b"))
    # bool predicates
    print("123".isdigit() and "abc".isalpha())


if __name__ == "__main__":
    main()
