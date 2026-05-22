def make_url(host: str, *, port: int = 80, scheme: str = "http") -> str:
    return f"{scheme}://{host}:{port}"


def main() -> None:
    print(make_url("example.com"))
    print(make_url("example.com", port=8080))
    print(make_url("example.com", scheme="https", port=443))


if __name__ == "__main__":
    main()
