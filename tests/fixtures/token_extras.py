import token


def main() -> None:
    print(token.LPAR)
    print(token.RPAR)
    print(token.PLUS)
    print(token.ISTERMINAL(1))
    print(token.ISEOF(0))


if __name__ == "__main__":
    main()
