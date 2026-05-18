def adder(n: int) -> int:
    def add(x: int) -> int:
        return x + n
    return add(10)


def counter_factory() -> int:
    def step(by: int) -> int:
        return by * 2
    a: int = step(3)
    b: int = step(5)
    return a + b


def main() -> None:
    print(adder(5))
    print(counter_factory())


if __name__ == "__main__":
    main()
