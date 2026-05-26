from email.mime.text import MIMEText


def main() -> None:
    # us-ascii avoids CPython's automatic base64 encoding of the payload
    # (gopy doesn't re-encode the body).
    msg = MIMEText("hello body", "plain", "us-ascii")
    print(msg.get("Content-Type"))
    print(msg.get("MIME-Version"))
    print(msg.get_payload())


if __name__ == "__main__":
    main()
