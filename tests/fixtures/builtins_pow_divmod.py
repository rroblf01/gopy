def main() -> None:
    q, r = divmod(17, 5)
    print(q, r)
    q2, r2 = divmod(20, 4)
    print(q2, r2)
    print(pow(2, 10))
    print(pow(3, 4, 5))
    print(pow(7, 13, 1000))
    print(pow(2, 20, 1000))
    words: list[str] = ["bb", "ccc", "a"]
    print(min(words, key=len))
    print(max(words, key=len))


if __name__ == "__main__":
    main()
