def sumall(*nums: int) -> int:
    total: int = 0
    for n in nums:
        total += n
    return total


def main() -> None:
    a: list[int] = [1, 2, 3, 4, 5]
    print(sumall(*a))
    print(sumall(10, 20, *a))


if __name__ == "__main__":
    main()
