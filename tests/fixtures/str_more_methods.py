def main() -> None:
    print("Hello WORLD".casefold())
    print("Hello WORLD".swapcase())
    print("abcXYZ".swapcase())
    print("a\tb\tc".expandtabs())
    print("a\tb\tc".expandtabs(4))
    print("abc\tdef\nx\ty".expandtabs(4))


if __name__ == "__main__":
    main()
