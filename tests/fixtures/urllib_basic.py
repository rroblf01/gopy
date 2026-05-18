from urllib.parse import quote, unquote


def main() -> None:
    s: str = "hello world & more"
    q: str = quote(s)
    print(q)
    print(unquote(q))


if __name__ == "__main__":
    main()
