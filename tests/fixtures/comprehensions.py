def main() -> None:
    nums: list[int] = [1, 2, 3, 4, 5]
    doubled: list[int] = [x * 2 for x in nums]
    for n in doubled:
        print(n)
    odd_squared: list[int] = [x * x for x in nums if x % 2 == 1]
    for n in odd_squared:
        print(n)
    names: list[str] = ["ada", "grace"]
    lengths: dict[str, int] = {n: len(n) for n in names}
    print(lengths["ada"])
    print(lengths["grace"])


if __name__ == "__main__":
    main()
