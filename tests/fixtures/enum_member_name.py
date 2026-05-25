from enum import Enum


class Color(Enum):
    RED = 1
    GREEN = 2
    BLUE = 3


def main() -> None:
    # Chained access on the class literal.
    print(Color.RED.name)
    print(Color.BLUE.name)

    # Variable typed as Color.
    c: Color = Color.GREEN
    print(c.name)


if __name__ == "__main__":
    main()
