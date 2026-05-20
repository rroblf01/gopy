import hmac


def main() -> None:
    h = hmac.new(b"secret", b"hello", "sha256")
    print(h.hexdigest())
    h2 = hmac.new(b"k", b"", "sha1")
    h2.update(b"hi")
    print(h2.hexdigest())
    print(hmac.compare_digest("abc", "abc"))
    print(hmac.compare_digest("abc", "abd"))


if __name__ == "__main__":
    main()
