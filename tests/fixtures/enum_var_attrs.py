from enum import Enum, auto


class Color(Enum):
    RED = auto()
    GREEN = auto()
    BLUE = auto()


def main() -> None:
    c = Color.RED
    print(c.name)
    print(c.value)
    print(c == Color.RED)
    print(c == Color.BLUE)

    g = Color.GREEN
    print(g.name)
    print(g.value)


if __name__ == "__main__":
    main()
