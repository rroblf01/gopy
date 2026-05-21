import keyword
import unicodedata


def main() -> None:
    print(keyword.iskeyword("if"))
    print(keyword.iskeyword("foo"))
    print(keyword.issoftkeyword("match"))
    print(keyword.issoftkeyword("xyz"))
    print(unicodedata.category("A"))
    print(unicodedata.category("a"))
    print(unicodedata.category("5"))
    print(unicodedata.category(" "))


if __name__ == "__main__":
    main()
