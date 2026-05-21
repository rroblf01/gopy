from email.utils import formatdate


def main() -> None:
    s: str = formatdate(0)
    print(len(s) > 20)
    print("1970" in s)


if __name__ == "__main__":
    main()
