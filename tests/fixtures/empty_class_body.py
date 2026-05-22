class Marker:
    """A doc string only."""


class Empty:
    pass


def main() -> None:
    m = Marker()
    e = Empty()
    print(type(m).__name__)
    print(type(e).__name__)


if __name__ == "__main__":
    main()
