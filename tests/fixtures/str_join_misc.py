def main() -> None:
    parts: list[str] = ["a", "b", "c"]
    print(",".join(parts))
    print("-".join([str(i) for i in range(5)]))
    words: list[str] = ["hello", "world"]
    print(" ".join(words))


if __name__ == "__main__":
    main()
