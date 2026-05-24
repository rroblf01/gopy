from typing import Self


class Counter:
    def __init__(self, n: int) -> None:
        self.n: int = n

    def add(self, k: int) -> Self:
        return Counter(self.n + k)

    def value(self) -> int:
        return self.n


def main() -> None:
    c = Counter(1)
    d = c.add(4)
    e = d.add(5)
    print(e.value())


if __name__ == "__main__":
    main()
