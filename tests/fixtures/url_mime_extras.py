from urllib.parse import urljoin, urlsplit
import mimetypes


def main() -> None:
    print(urljoin("https://example.com/a/b/", "c"))
    print(urljoin("https://example.com/a/b/c", "../d"))
    print(urljoin("https://example.com/", "https://other.com/x"))

    parts = urlsplit("https://user@example.com:8080/p?q=1#frag")
    print(parts.scheme)
    print(parts.path)

    mime_type = mimetypes.guess_type("foo.html")
    print(mime_type[0])
    txt = mimetypes.guess_type("notes.txt")
    print(txt[0])
    unknown = mimetypes.guess_type("file.weirdtype9999")
    print(unknown[0] in (None, ""))


if __name__ == "__main__":
    main()
