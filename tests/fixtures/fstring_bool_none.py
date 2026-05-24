def main() -> None:
    b = True
    print(f"flag={b}")
    c = False
    print(f"c={c}, b={b}")
    print(f"both: {b} and {c}")
    # bool from comparison
    x = 5
    print(f"big? {x > 3}")
    # None via assignment - skip, hard to type
    # mixed
    s = "x"
    n = 42
    print(f"{s} {n} {b}")


if __name__ == "__main__":
    main()
