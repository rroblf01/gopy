def main() -> None:
    nums: list[int] = [1, 2, 3, 4]
    squared = {n * n for n in nums if n > 1}
    print(len(squared))
    out: list[int] = list(squared)
    out.sort()
    for v in out:
        print(v)


if __name__ == "__main__":
    main()
