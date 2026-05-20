def main() -> None:
    table = str.maketrans("abc", "xyz")
    print("abcabc".translate(table))
    print("alphabet".translate(table))
    drop = str.maketrans("", "", "aeiou")
    print("hello world".translate(drop))


if __name__ == "__main__":
    main()
