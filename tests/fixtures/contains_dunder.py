class Range:
    lo: int
    hi: int

    def __init__(self, lo: int, hi: int) -> None:
        self.lo = lo
        self.hi = hi

    def __contains__(self, x: int) -> bool:
        return self.lo <= x and x < self.hi


def main() -> None:
    r = Range(0, 10)
    print(5 in r)
    print(10 in r)
    print(-1 not in r)
    print(0 in r)


if __name__ == "__main__":
    main()
