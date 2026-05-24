def main() -> None:
    s = "abcdef"
    print(s[2:])
    print(s[:3])
    print(s[1:-1])
    print(s[-3:])
    print(s[:-2])
    print(s[3:3])
    print(s[10:20])
    print(s[100:])
    print(s[:0])
    print(s[::-1])
    # via variable bounds
    lo = 1
    hi = 4
    print(s[lo:hi])
    # arithmetic bounds
    print(s[len(s) - 3:])


if __name__ == "__main__":
    main()
