class Math:
    @staticmethod
    def add(a: int, b: int) -> int:
        return a + b

    @staticmethod
    def mul(a: int, b: int) -> int:
        return a * b

    @staticmethod
    def expr(x: int) -> int:
        return Math.add(Math.mul(x, 3), Math.mul(x, 2))


def main() -> None:
    print(Math.add(2, 3))
    print(Math.mul(4, 5))
    print(Math.expr(7))


if __name__ == "__main__":
    main()
