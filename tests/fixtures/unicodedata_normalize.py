import unicodedata


def main() -> None:
    s = "cafe"
    print(unicodedata.normalize("NFC", s))


if __name__ == "__main__":
    main()
