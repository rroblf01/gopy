def main() -> None:
    print("  hi there  ".lstrip())
    print("  hi there  ".rstrip())
    print("xxhellox".lstrip("x"))
    print("xxhelloxx".rstrip("x"))
    print("aabbaabbcc".count("aa"))
    print("hello world".title())
    print("hello".capitalize())
    print("hi".center(7, "*"))
    print("hi".ljust(5, "."))
    print("hi".rjust(5, "."))
    print("42".zfill(6))
    print("-7".zfill(5))


if __name__ == "__main__":
    main()
