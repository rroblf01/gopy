import secrets


def main() -> None:
    h = secrets.token_hex(16)
    print(len(h))
    u = secrets.token_urlsafe(16)
    print(len(u) > 0)
    h2 = secrets.token_hex()
    print(len(h2))


if __name__ == "__main__":
    main()
