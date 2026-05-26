import base64


def main() -> None:
    enc = base64.standard_b64encode(b"hi")
    if not isinstance(enc, str):
        enc = enc.decode()
    print(enc)


if __name__ == "__main__":
    main()
