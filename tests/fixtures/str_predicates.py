def main() -> None:
    print("123".isdigit())
    print("12a".isdigit())
    print("".isdigit())
    print("abc".isalpha())
    print("ab1".isalpha())
    print("Abc123".isalnum())
    print("Abc 123".isalnum())
    print("   ".isspace())
    print("  a".isspace())
    print("HELLO".isupper())
    print("Hello".isupper())
    print("HELLO!".isupper())
    print("123".isupper())
    print("hello".islower())
    print("Hello".islower())
    print("hello!".islower())


if __name__ == "__main__":
    main()
