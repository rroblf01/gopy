def main() -> None:
    a = [1, 2, 3]
    b = [4, 5, 6]
    merged = [*a, *b]
    print(merged)
    appended = [*a, 99]
    print(appended)
    prepended = [0, *b]
    print(prepended)


if __name__ == "__main__":
    main()
