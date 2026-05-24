def main() -> None:
    def split_first_rest(xs: list[int]) -> tuple[int, list[int]]:
        return xs[0], xs[1:]

    head, tail = split_first_rest([10, 20, 30, 40])
    print(head)
    print(tail)

    def minmax(xs: list[int]) -> tuple[int, int]:
        return min(xs), max(xs)

    lo, hi = minmax([3, 1, 4, 1, 5, 9, 2])
    print(lo, hi)

    def make_pair(n: int) -> tuple[int, int]:
        return n, n * 2

    a, b = make_pair(5)
    print(a, b)


if __name__ == "__main__":
    main()
