def total(*nums: int) -> int:
    s: int = 0
    for n in nums:
        s += n
    return s


def main() -> None:
    print(total(1, 2, 3))
    print(total())
    print(total(10, 20))


if __name__ == "__main__":
    main()
