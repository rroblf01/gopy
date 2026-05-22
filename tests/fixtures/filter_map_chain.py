def main() -> None:
    nums: list[int] = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    evens = list(filter(lambda x: x % 2 == 0, nums))
    print(evens)
    doubled = list(map(lambda x: x * 2, nums))
    print(doubled)
    big: list[int] = list(filter(lambda x: x > 5, nums))
    tripled = list(map(lambda x: x * 3, big))
    print(tripled)


if __name__ == "__main__":
    main()
