import hashlib


def main() -> None:
    dk = hashlib.pbkdf2_hmac("sha256", b"password", b"salt", 1000, 32)
    print(len(dk) > 0)


if __name__ == "__main__":
    main()
