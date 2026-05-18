from collections import Counter


def main() -> None:
    items: list[str] = ["a", "b", "a", "c", "b", "a"]
    counts: dict[str, int] = Counter(items)
    # Order-independent assertion: pull each known key.
    print(counts["a"])
    print(counts["b"])
    print(counts["c"])
    nums: list[int] = [1, 1, 2, 3, 3, 3]
    nc: dict[int, int] = Counter(nums)
    print(nc[1])
    print(nc[2])
    print(nc[3])


if __name__ == "__main__":
    main()
