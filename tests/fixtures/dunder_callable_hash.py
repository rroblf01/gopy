class Adder:
    def __init__(self, k: int) -> None:
        self.k = k

    def __call__(self, x: int) -> int:
        return x + self.k

    def __hash__(self) -> int:
        return self.k * 1000 + 7


def main() -> None:
    a = Adder(10)
    print(a(5))
    print(a(100))
    b = Adder(3)
    print(b(4))
    print(hash(a))
    print(hash(b))


if __name__ == "__main__":
    main()
