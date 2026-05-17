def main() -> None:
    nums: list[int] = [10, 20, 30, 40, 50]
    print(len(nums[1:3]))
    for v in nums[1:3]:
        print(v)
    for v in nums[:2]:
        print(v)
    for v in nums[3:]:
        print(v)
    for v in nums[:]:
        print(v)


if __name__ == "__main__":
    main()
