class Counter:
    n: int

    def __init__(self) -> None:
        self.n = 0

    def inc(self) -> None:
        self.n += 1

    def __bool__(self) -> bool:
        return self.n > 0


def main() -> None:
    c = Counter()
    if c:
        print("starts truthy")
    else:
        print("starts falsy")
    c.inc()
    c.inc()
    if c:
        print("now truthy")
    if not c:
        print("never")
    # in conditional expression
    msg = "filled" if c else "empty"
    print(msg)


if __name__ == "__main__":
    main()
