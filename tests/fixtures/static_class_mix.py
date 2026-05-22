class Calc:
    @staticmethod
    def double(x: int) -> int:
        return x * 2

    @staticmethod
    def triple(x: int) -> int:
        return x * 3

    @classmethod
    def six_times(cls, x: int) -> int:
        return cls.double(cls.triple(x))


def main() -> None:
    print(Calc.double(5))
    print(Calc.triple(5))
    print(Calc.six_times(5))


if __name__ == "__main__":
    main()
