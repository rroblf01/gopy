from email import utils as eu


def main() -> None:
    name, addr = eu.parseaddr("Foo Bar <foo@example.com>")
    print(name)
    print(addr)
    print(eu.formataddr(["Alice", "alice@example.com"]))
    msg = eu.make_msgid()
    print(len(msg) > 5)


if __name__ == "__main__":
    main()
