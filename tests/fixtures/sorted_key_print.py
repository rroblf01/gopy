def main() -> None:
    words: list[str] = ["banana", "apple", "cherry"]
    sorted_words = sorted(words, key=len)
    print(sorted_words)
    nums: list[int] = [-3, 1, -2, 5]
    sorted_abs = sorted(nums, key=abs)
    print(sorted_abs)
    print(sorted(nums, reverse=True))


if __name__ == "__main__":
    main()
