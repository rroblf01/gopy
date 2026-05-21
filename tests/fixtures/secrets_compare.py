import secrets


def main() -> None:
    print(secrets.compare_digest("token", "token"))
    print(secrets.compare_digest("token", "other"))
    print(secrets.compare_digest("", ""))


if __name__ == "__main__":
    main()
