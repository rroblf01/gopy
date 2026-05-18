from functools import reduce


def main() -> None:
    nums: list[int] = [1, 2, 3, 4, 5]
    total: int = reduce(lambda a, b: a + b, nums)
    print(total)
    product: int = reduce(lambda a, b: a * b, nums, 1)
    print(product)
    maxv: int = reduce(lambda a, b: a if a > b else b, nums)
    print(maxv)


if __name__ == "__main__":
    main()
