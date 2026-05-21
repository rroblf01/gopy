import hashlib


def main() -> None:
    h = hashlib.new("sha256", b"hello")
    print(len(h.hexdigest()) == 64)
    h2 = hashlib.new("md5", b"hello")
    print(len(h2.hexdigest()) == 32)


if __name__ == "__main__":
    main()
