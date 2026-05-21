class Box:
    def __init__(self, n: int) -> None:
        self.n = n

    def __str__(self) -> str:
        return f"Box[{self.n}]"

    def __int__(self) -> int:
        return self.n

    def __float__(self) -> float:
        return float(self.n) + 0.5

    def __reversed__(self) -> list[int]:
        out: list[int] = []
        i = self.n
        while i > 0:
            out.append(i)
            i = i - 1
        return out


def main() -> None:
    b = Box(3)
    print(str(b))
    print(int(b))
    print(float(b))
    for v in reversed(b):
        print(v)


if __name__ == "__main__":
    main()
