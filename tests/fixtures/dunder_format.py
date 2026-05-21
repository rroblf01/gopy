class Money:
    def __init__(self, cents: int) -> None:
        self.cents = cents

    def __format__(self, spec: str) -> str:
        d: int = int(self.cents / 100)
        c: int = self.cents - d * 100
        if spec == "short":
            return f"${d}"
        if spec == "long":
            return f"${d}.{c:02d}"
        return f"{self.cents}c"


def main() -> None:
    m = Money(1234)
    print(f"[{m}]")
    print(f"[{m:short}]")
    print(f"[{m:long}]")
    print(f"[{m:other}]")


if __name__ == "__main__":
    main()
