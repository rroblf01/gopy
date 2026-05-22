def main() -> None:
    nums: list[int] = [3, 1, 4, 1, 5, 9, 2, 6]
    nums.sort()
    print(nums)
    nums.reverse()
    print(nums)
    counts: dict[str, int] = {}
    words = ["apple", "banana", "apple", "cherry", "banana", "apple"]
    for w in words:
        counts[w] = counts.get(w, 0) + 1
    for k in sorted(counts.keys()):
        print(k, counts[k])


if __name__ == "__main__":
    main()
