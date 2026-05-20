import string


def main() -> None:
    print(string.capwords("hello world"))
    print(string.capwords("  HELLO   wOrLd  "))
    print(string.capwords("foo-bar-baz", "-"))


if __name__ == "__main__":
    main()
