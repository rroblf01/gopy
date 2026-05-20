from urllib.parse import quote_plus, unquote_plus


def main() -> None:
    print(quote_plus("hello world"))
    print(quote_plus("a&b=c"))
    print(unquote_plus("hello+world"))
    print(unquote_plus("a%26b%3Dc"))


if __name__ == "__main__":
    main()
