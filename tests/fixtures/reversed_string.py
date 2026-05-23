def main() -> None:
    s = "hello"
    chars: list[str] = list(s)
    rev = "".join(reversed(chars))
    print(rev)


if __name__ == "__main__":
    main()
