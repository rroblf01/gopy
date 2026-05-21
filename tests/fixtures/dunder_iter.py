class Sequence:
    def __init__(self, n: int) -> None:
        self.n = n

    def __iter__(self) -> list[int]:
        out: list[int] = []
        i: int = 0
        while i < self.n:
            out.append(i * 2)
            i = i + 1
        return iter(out)


def main() -> None:
    s = Sequence(5)
    for v in s:
        print(v)
    total: int = 0
    for v in s:
        total = total + v
    print(total)


if __name__ == "__main__":
    main()
