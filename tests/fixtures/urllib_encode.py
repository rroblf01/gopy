from urllib.parse import urlencode


def main() -> None:
    one: dict[str, str] = {"q": "hello world"}
    print(urlencode(one))


if __name__ == "__main__":
    main()
