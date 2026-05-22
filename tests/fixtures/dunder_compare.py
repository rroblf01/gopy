class Money:
    def __init__(self, amount: int) -> None:
        self.amount = amount

    def __eq__(self, other: "Money") -> bool:
        return self.amount == other.amount

    def __lt__(self, other: "Money") -> bool:
        return self.amount < other.amount

    def __hash__(self) -> int:
        return self.amount


def main() -> None:
    a = Money(10)
    b = Money(10)
    c = Money(20)
    print(a == b)
    print(a == c)
    print(a < c)
    print(c < a)


if __name__ == "__main__":
    main()
