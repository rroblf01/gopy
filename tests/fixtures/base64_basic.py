import base64


def main() -> None:
    # b64encode in CPython returns bytes; the Go shim returns str. Calling
    # .decode() on the result is a no-op in our shim and a real conversion
    # in CPython — both end up with the same printable string.
    enc: str = base64.b64encode("hello world".encode()).decode()
    print(enc)
    dec: str = base64.b64decode(enc).decode()
    print(dec)


if __name__ == "__main__":
    main()
