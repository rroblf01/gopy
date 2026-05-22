import hashlib


def main() -> None:
    h = hashlib.sha224(b"hello world")
    print(h.hexdigest())
    h2 = hashlib.sha384(b"hello world")
    print(h2.hexdigest())


if __name__ == "__main__":
    main()
