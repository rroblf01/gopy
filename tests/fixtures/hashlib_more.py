import hashlib


def main() -> None:
    print(hashlib.sha1(b"hello").hexdigest())
    print(hashlib.sha512(b"hello").hexdigest())
    print(hashlib.sha256(b"hello").hexdigest())
    print(hashlib.md5(b"hello").hexdigest())


if __name__ == "__main__":
    main()
