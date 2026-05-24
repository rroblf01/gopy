def main() -> None:
    print(repr("ab" * -3))
    print(repr("ab" * 0))
    print(repr("ab" * 1))
    print(repr("ab" * 3))
    print(repr(-3 * "x"))
    # variable count
    n = 0
    print(repr("hi" * n))
    n = 4
    print(repr("hi" * n))
    n = -5
    print(repr("hi" * n))


if __name__ == "__main__":
    main()
