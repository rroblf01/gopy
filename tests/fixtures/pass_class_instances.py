class Greeter:
    def __init__(self, name: str) -> None:
        self.name = name

    def hi(self) -> str:
        return f"hello {self.name}"


def call_hi(g: Greeter) -> str:
    return g.hi()


def main() -> None:
    a = Greeter("alice")
    b = Greeter("bob")
    print(call_hi(a))
    print(call_hi(b))
    greeters: list[Greeter] = [a, b, Greeter("carol")]
    for g in greeters:
        print(call_hi(g))


if __name__ == "__main__":
    main()
