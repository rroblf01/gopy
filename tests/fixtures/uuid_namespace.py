import uuid


def main() -> None:
    print(uuid.NAMESPACE_DNS)
    print(str(uuid.uuid3(uuid.NAMESPACE_DNS, "example.com")))
    print(str(uuid.uuid5(uuid.NAMESPACE_DNS, "example.com")))
    print(len(str(uuid.uuid4())) == 36)


if __name__ == "__main__":
    main()
