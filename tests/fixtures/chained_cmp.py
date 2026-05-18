def in_range(n: int, lo: int, hi: int) -> bool:
    return lo <= n <= hi


def main() -> None:
    print(in_range(5, 0, 10))
    print(in_range(0, 0, 10))
    print(in_range(10, 0, 10))
    print(in_range(11, 0, 10))
    print(in_range(-1, 0, 10))


if __name__ == "__main__":
    main()
