class Counter:
    def __init__(self, base: int) -> None:
        self.base = base

    def __getitem__(self, key: int) -> int:
        return self.base + key

    def __setitem__(self, key: int, value: int) -> None:
        self.base = value

    def __contains__(self, key: int) -> bool:
        return key >= 0 and key < 10


def main() -> None:
    b = Counter(100)
    print(b[5])
    b[0] = 200
    print(b[5])
    print(3 in b)
    print(99 in b)
    print(99 not in b)


if __name__ == "__main__":
    main()
