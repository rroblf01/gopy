def main() -> None:
    words: list[str] = ["banana", "apple", "cherry"]
    sorted_words = sorted(words, key=lambda w: len(w))
    print(sorted_words)
    nums: list[int] = [-3, 1, -2, 5]
    sorted_abs = sorted(nums, key=lambda x: x if x >= 0 else -x)
    print(sorted_abs)
    print(sorted(nums, reverse=True))


if __name__ == "__main__":
    main()
