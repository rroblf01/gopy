def main() -> None:
    nums: list[int] = [1, 2, 3, 4, 5]
    doubled: list[int] = map(lambda x: x * 2, nums)
    for v in doubled:
        print(v)

    evens: list[int] = filter(lambda x: x % 2 == 0, nums)
    for v in evens:
        print(v)

    desc: list[int] = sorted(nums, key=lambda x: -x)
    for v in desc:
        print(v)

    rev: list[int] = sorted(nums, reverse=True)
    for v in rev:
        print(v)


if __name__ == "__main__":
    main()
