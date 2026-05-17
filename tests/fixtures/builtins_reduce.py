def main() -> None:
    nums: list[int] = [3, 1, 4, 1, 5, 9, 2, 6]
    print(sum(nums))
    print(min(nums))
    print(max(nums))
    flags: list[bool] = [True, True, False]
    print(any(flags))
    print(all(flags))
    print(any([False, False]))
    print(all([True, True]))
    s: list[int] = sorted(nums)
    for v in s:
        print(v)


if __name__ == "__main__":
    main()
