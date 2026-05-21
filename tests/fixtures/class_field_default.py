class Config:
    name: str = "default"
    timeout: int = 30
    debug: bool = False


class WithInit:
    n: int = 10
    label: str = "hi"

    def __init__(self, n: int) -> None:
        self.n = n


class Override:
    base: int = 100

    def __init__(self) -> None:
        self.base = self.base + 5


def main() -> None:
    c = Config()
    print(c.name)
    print(c.timeout)
    print(c.debug)

    w = WithInit(42)
    print(w.n)
    print(w.label)

    o = Override()
    print(o.base)


if __name__ == "__main__":
    main()
