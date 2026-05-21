class Foo:
    def __init__(self, n: int) -> None:
        self.n = n


def main() -> None:
    f = Foo(7)
    print(f.__class__.__name__)
    print(Foo.__name__)
    print(type(f).__name__)


if __name__ == "__main__":
    main()
