def main() -> None:
    for i in range(0, 20, 3):
        print(i)
    for i in range(10, 0, -2):
        print(i)
    total: int = 0
    for n in range(1, 11):
        total += n
    print(total)
    nums: list[int] = list(range(5))
    print(nums)


if __name__ == "__main__":
    main()
