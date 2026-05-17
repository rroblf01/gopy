def main() -> None:
    nums: list[int] = [1, 2, 3, 4, 5]
    total: int = 0
    for n in nums:
        total += n
    print(total)
    print(len(nums))
    nums.append(6)
    print(len(nums))
    print(nums[5])


if __name__ == "__main__":
    main()
