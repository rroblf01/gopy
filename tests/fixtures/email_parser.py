from email.parser import Parser


def main() -> None:
    src = "From: a@example.com\r\nTo: b@example.com\r\nSubject: Hi\r\n\r\nHello world"
    p = Parser()
    msg = p.parsestr(src)
    print(msg.get("from"))
    print(msg.get("to"))
    print(msg.get("subject"))
    print(msg.get_payload())


if __name__ == "__main__":
    main()
