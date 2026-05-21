class Box:
    def __init__(self, n: int) -> None:
        self.n = n


def describe(x: object) -> str:
    match x:
        case int():
            return "int"
        case str():
            return "str"
        case Box(n=5) as b:
            return f"box5={b.n}"
        case Box() as b:
            return f"box={b.n}"
        case other:
            return f"unknown: {other}"


def main() -> None:
    print(describe(42))
    print(describe("hello"))
    print(describe(Box(5)))
    print(describe(Box(99)))
    print(describe(3.14))


if __name__ == "__main__":
    main()
