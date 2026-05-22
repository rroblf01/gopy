def main() -> None:
    s = "  Hello World  "
    print(s.strip().lower().replace("hello", "hi"))
    parts = "a,b,c,d".split(",")
    print("-".join(parts))
    nums: list[int] = [3, 1, 4, 1, 5, 9, 2, 6]
    print(sorted(nums)[:3])
    print(sorted(nums, reverse=True)[:3])


if __name__ == "__main__":
    main()
