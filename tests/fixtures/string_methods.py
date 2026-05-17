def main() -> None:
    s: str = "  Hello World  "
    print(s.upper())
    print(s.lower())
    print(s.strip())
    print(s.strip().replace("World", "Go"))
    print(s.strip().startswith("Hello"))
    print(s.strip().endswith("World"))
    print(s.strip().find("World"))
    print(s.strip().find("zzz"))
    parts: list[str] = "a,b,c,d".split(",")
    print(len(parts))
    joined: str = "-".join(parts)
    print(joined)
    words: list[str] = "alpha beta gamma".split()
    print(len(words))


if __name__ == "__main__":
    main()
