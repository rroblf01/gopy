def main() -> None:
    nums: list[int] = [3, 1, 4, 1, 5]
    print(min(nums, default=99))
    print(max(nums, default=-1))
    print(min([], default=99))
    print(max([], default=-1))
    print(min([], default=0))
    print(max([], default=0))
    print(sum(x * x for x in range(5)))
    print(sum(x for x in range(10) if x % 2 == 0))


if __name__ == "__main__":
    main()
