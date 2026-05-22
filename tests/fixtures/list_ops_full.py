def main() -> None:
    nums: list[int] = [3, 1, 4, 1, 5, 9, 2, 6, 5, 3]
    nums.sort()
    print(nums)
    # find min/max
    print(min(nums))
    print(max(nums))
    print(sum(nums))
    print(len(nums))
    # count items
    print(nums.count(1))
    print(nums.count(5))
    # check membership
    print(9 in nums)
    print(99 in nums)
    # index
    print(nums.index(5))
    # remove duplicates by passing through set
    unique: list[int] = sorted(set(nums))
    print(unique)


if __name__ == "__main__":
    main()
