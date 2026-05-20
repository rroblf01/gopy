import uuid


def main() -> None:
    u = uuid.uuid4()
    s = str(u)
    print(len(s))
    print(s[8:9])
    print(s[13:14])
    print(s[14:15])
    print(s[18:19])
    print(s[23:24])


if __name__ == "__main__":
    main()
