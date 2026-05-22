from functools import reduce


def main() -> None:
    nums: list[int] = [1, 2, 3, 4, 5]
    s = reduce(lambda a, b: a + b, nums)
    print(s)
    m = reduce(lambda a, b: a * b, nums, 1)
    print(m)
    biggest = reduce(lambda a, b: a if a > b else b, nums)
    print(biggest)


if __name__ == "__main__":
    main()
