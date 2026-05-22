def main() -> None:
    s = "Hello"
    print(s.isdigit())
    print("12345".isdigit())
    print(s.isalpha())
    print(s.isalnum())
    print("hello".islower())
    print(s.isupper())
    print("HELLO".isupper())
    print(s.swapcase())
    print(s.casefold())


if __name__ == "__main__":
    main()
