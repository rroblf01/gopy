import uuid


def main() -> None:
    u = uuid.UUID("12345678-1234-5678-9234-567812345678")
    print(u.hex)
    print(u.version)


if __name__ == "__main__":
    main()
