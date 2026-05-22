def main() -> None:
    nums: list[int] = [10, 20, 30]
    for i, n in enumerate(nums):
        print(i, n)
    items: list[str] = ["a", "b", "c"]
    for i, c in enumerate(items, start=1):
        print(i, c)


if __name__ == "__main__":
    main()
