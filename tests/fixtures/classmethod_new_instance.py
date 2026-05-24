from __future__ import annotations


class Box:
    def __init__(self, n: int) -> None:
        self.n: int = n

    def doubled(self) -> Box:
        return Box(self.n * 2)

    def tripled(self) -> "Box":
        return Box(self.n * 3)


def main() -> None:
    b = Box(3)
    d = b.doubled()
    print(d.n)
    t = b.tripled()
    print(t.n)


if __name__ == "__main__":
    main()
