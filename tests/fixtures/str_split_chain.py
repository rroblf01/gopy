def main() -> None:
    s = "Hello, World!"
    print(s.replace(",", "").upper().split())
    text = "a b c d e"
    print(text.split()[2])
    parts = text.split()
    parts.reverse()
    print(parts)
    print(",".join(reversed(text.split())))


if __name__ == "__main__":
    main()
