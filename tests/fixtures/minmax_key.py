def main() -> None:
    words: list[str] = ["foo", "ab", "elephant", "xyz"]
    print(min(words, key=lambda s: len(s)))
    print(max(words, key=lambda s: len(s)))
    nums: list[int] = [3, -7, 2, -10, 5]
    print(max(nums, key=lambda n: -n))
    print(min(nums, key=lambda n: -n))


if __name__ == "__main__":
    main()
