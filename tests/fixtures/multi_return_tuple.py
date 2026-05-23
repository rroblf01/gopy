def stats(nums: list[int]) -> tuple[int, int, int]:
    return (min(nums), max(nums), sum(nums))


def divmod_pair(a: int, b: int) -> tuple[int, int]:
    return (a // b, a % b)


def main() -> None:
    lo, hi, tot = stats([3, 1, 4, 1, 5, 9, 2, 6])
    print(lo, hi, tot)
    q, r = divmod_pair(17, 5)
    print(q, r)


if __name__ == "__main__":
    main()
