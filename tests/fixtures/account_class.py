class Account:
    balance: float

    def __init__(self, initial: float) -> None:
        self.balance = initial

    def deposit(self, amount: float) -> None:
        self.balance += amount

    def withdraw(self, amount: float) -> bool:
        if amount > self.balance:
            return False
        self.balance -= amount
        return True

    def __str__(self) -> str:
        return f"Account(${self.balance:.2f})"


def main() -> None:
    a = Account(100.0)
    print(a)
    a.deposit(50.5)
    print(a)
    ok = a.withdraw(200.0)
    print(ok)
    ok = a.withdraw(75.25)
    print(ok)
    print(a)


if __name__ == "__main__":
    main()
