from enum import Enum, auto


class Color(Enum):
    RED = auto()
    GREEN = auto()
    BLUE = auto()


class Mixed(Enum):
    A = 10
    B = auto()
    C = auto()
    D = 100
    E = auto()


def main() -> None:
    print(Color.RED == Color.RED)
    print(Color.RED == Color.GREEN)
    print(Mixed.A == Mixed.A)
    print(Mixed.B == Mixed.A)


if __name__ == "__main__":
    main()
