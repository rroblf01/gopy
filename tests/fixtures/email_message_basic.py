from email.message import EmailMessage


def main() -> None:
    m = EmailMessage()
    m.add_header("From", "alice@example.com")
    m.add_header("Subject", "hi")
    m.set_payload("hello world")
    print(m.get("From"))
    print(m.get("Subject"))
    print(m.get_payload())


if __name__ == "__main__":
    main()
