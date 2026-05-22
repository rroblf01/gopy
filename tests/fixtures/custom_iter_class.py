class Range:
    def __init__(self, start: int, stop: int) -> None:
        self.start = start
        self.stop = stop

    def __iter__(self) -> list[int]:
        out: list[int] = []
        i = self.start
        while i < self.stop:
            out.append(i)
            i += 1
        return iter(out)


def main() -> None:
    r = Range(5, 10)
    for v in r:
        print(v)
    total: int = 0
    for v in r:
        total += v
    print(total)


if __name__ == "__main__":
    main()
