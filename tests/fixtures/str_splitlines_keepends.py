def main() -> None:
    s = "a\nb\r\nc\rd"
    print(s.splitlines())
    print(s.splitlines(False))
    print(s.splitlines(True))


if __name__ == "__main__":
    main()
