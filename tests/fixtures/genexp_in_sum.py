def main() -> None:
    nums: list[int] = [1, 2, 3, 4, 5]
    doubled = [x * 2 for x in nums if x > 2]
    print(doubled)
    summed = sum(x for x in nums if x % 2 == 0)
    print(summed)
    most = max(x for x in nums)
    print(most)
    least = min(x * 3 for x in nums if x < 4)
    print(least)


if __name__ == "__main__":
    main()
