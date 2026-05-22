def main() -> None:
    a: set[int] = {3, 1, 2}
    print(sorted(a))
    print(len(a))
    print(2 in a)
    print(5 in a)


if __name__ == "__main__":
    main()
