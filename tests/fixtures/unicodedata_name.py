import unicodedata


def main() -> None:
    print(unicodedata.name("A"))
    print(unicodedata.name("z"))
    print(unicodedata.name("5"))
    print(unicodedata.name("@"))
    print(unicodedata.name("?", "unknown"))


if __name__ == "__main__":
    main()
