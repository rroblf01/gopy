import hashlib


def main() -> None:
    # .encode() is a no-op in the gopy shim — both runtimes consume str.
    print(hashlib.sha256("hello".encode()).hexdigest())
    print(hashlib.md5("world".encode()).hexdigest())


if __name__ == "__main__":
    main()
