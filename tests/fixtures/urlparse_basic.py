from urllib.parse import urlparse


def main() -> None:
    u = urlparse("https://example.com:8080/path/to/page?q=hello#top")
    print(u.scheme)
    print(u.netloc)
    print(u.path)
    print(u.query)
    print(u.fragment)


if __name__ == "__main__":
    main()
