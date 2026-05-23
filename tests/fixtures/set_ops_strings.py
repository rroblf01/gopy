def main() -> None:
    a: set[str] = {"apple", "banana", "cherry"}
    b: set[str] = {"banana", "cherry", "date"}
    print(sorted(a & b))
    print(sorted(a | b))
    print(sorted(a - b))
    print(sorted(a ^ b))
    # Membership
    print("apple" in a)
    print("date" in a)


if __name__ == "__main__":
    main()
