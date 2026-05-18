from enum import Enum


class Color(Enum):
    RED = 1
    GREEN = 2
    BLUE = 3


def describe(c: Color) -> str:
    if c == Color.RED:
        return "red"
    if c == Color.GREEN:
        return "green"
    return "blue"


def main() -> None:
    print(describe(Color.RED))
    print(describe(Color.GREEN))
    print(describe(Color.BLUE))


if __name__ == "__main__":
    main()
