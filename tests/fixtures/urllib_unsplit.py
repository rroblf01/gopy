from urllib.parse import urlunsplit, urlunparse


def main() -> None:
    out: str = urlunsplit(["https", "example.com", "/path", "k=v", "frag"])
    print(out)
    out2: str = urlunparse(["https", "example.com", "/path", "p", "k=v", "frag"])
    print(out2)


if __name__ == "__main__":
    main()
