def main() -> None:
    s = "  Hello, World!  "
    print(s.strip())
    print(s.strip().lower())
    print(s.lstrip())
    print(s.rstrip())
    parts = s.strip().split(", ")
    print(parts)
    csv = ",".join(["a", "b", "c"])
    print(csv)
    print(csv.replace(",", " | "))
    text = "abc-def-ghi"
    print(text.count("-"))
    print(text.startswith("abc"))
    print(text.endswith("ghi"))
    print(text.upper())
    print(len(text))


if __name__ == "__main__":
    main()
