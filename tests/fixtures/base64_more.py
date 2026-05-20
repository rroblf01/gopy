import base64


def main() -> None:
    print(base64.urlsafe_b64encode(b"hello world").decode())
    print(base64.urlsafe_b64decode("aGVsbG8gd29ybGQ=").decode())
    print(base64.b32encode(b"hello").decode())
    print(base64.b32decode("NBSWY3DP").decode())
    print(base64.b16encode(b"hello").decode())
    print(base64.b16decode("68656C6C6F").decode())


if __name__ == "__main__":
    main()
