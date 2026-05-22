class Wrapped:
    def __init__(self, n: int) -> None:
        self.n = n

    def __index__(self) -> int:
        return self.n


def main() -> None:
    arr = [10, 20, 30, 40, 50]
    w = Wrapped(2)
    print(arr[w.__index__()])
    print(hex(w.__index__()))


if __name__ == "__main__":
    main()
