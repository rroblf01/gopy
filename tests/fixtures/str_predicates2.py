def main() -> None:
    print("foo".isidentifier())
    print("_foo".isidentifier())
    print("9foo".isidentifier())
    print("foo bar".isidentifier())
    print("".isidentifier())
    print("hello".isprintable())
    print("hello\nworld".isprintable())
    print("".isprintable())
    print("hello".isascii())
    print("café".isascii())
    print("".isascii())
    print("123".isnumeric())
    print("abc".isnumeric())
    print("123".isdecimal())


if __name__ == "__main__":
    main()
