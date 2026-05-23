def main() -> None:
    data: list[int] = [1, 4, 9, 16, 25]
    out: list[int] = [y for x in data if (y := x * 2) > 5]
    print(out)
    # Walrus in if condition
    nums: list[int] = [1, 5, 10, 15, 20, 25]
    if (total := sum(nums)) > 50:
        print(f"sum is {total}")


if __name__ == "__main__":
    main()
