import unicodedata


def main() -> None:
    print(unicodedata.bidirectional("5"))
    print(unicodedata.mirrored("("))
    print(unicodedata.combining("A"))


if __name__ == "__main__":
    main()
