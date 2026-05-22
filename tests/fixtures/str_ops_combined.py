def main() -> None:
    s = "Hello, World!"
    print(s.lower())
    print(s.upper())
    print(s.replace("World", "Python"))
    print(s.split(", "))
    parts = ["a", "b", "c"]
    print(",".join(parts))
    print(len(s))
    print(s[0:5])
    print(s[-6:-1])
    print(s.startswith("Hello"))
    print(s.endswith("!"))
    print("o" in s)


if __name__ == "__main__":
    main()
