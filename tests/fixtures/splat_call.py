def summing(*nums: int) -> int:
    total: int = 0
    for n in nums:
        total += int(n)
    return total


def main() -> None:
    print(summing(1, 2, 3))
    vals: list[int] = [10, 20, 30, 40]
    print(summing(*vals))


if __name__ == "__main__":
    main()
