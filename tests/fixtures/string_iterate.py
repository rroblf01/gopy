def main() -> None:
    s = "hello"
    for c in s:
        print(c)
    chars: list[str] = list(s)
    print(chars)
    print(len(chars))
    upper_chars: list[str] = []
    for c in s:
        upper_chars.append(c.upper())
    print(upper_chars)


if __name__ == "__main__":
    main()
