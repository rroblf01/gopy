import html.entities


def main() -> None:
    n2c: dict[str, int] = html.entities.name2codepoint
    print(n2c["amp"])
    print(n2c["lt"])
    print(n2c["gt"])


if __name__ == "__main__":
    main()
