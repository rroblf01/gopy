def main() -> None:
    s = "  Hello-World!  "
    # Chained string methods
    out = s.strip().lower().replace("-", "_")
    print(out)
    # split-then-process
    parts = "a,b,c,d".split(",")
    upper_parts: list[str] = []
    for p in parts:
        upper_parts.append(p.upper())
    print(upper_parts)
    # title and capitalize
    print("hello world".title())
    print("hello".capitalize())
    print("HELLO".swapcase())


if __name__ == "__main__":
    main()
