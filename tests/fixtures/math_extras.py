import math


def main() -> None:
    print(math.dist([0.0, 0.0], [3.0, 4.0]))
    print(round(math.dist([1.0, 2.0, 3.0], [4.0, 6.0, 3.0]), 4))
    print(math.prod([1, 2, 3, 4]))
    print(math.prod([5]))
    print(math.remainder(10.0, 3.0))
    print(math.remainder(10.5, 3.0))


if __name__ == "__main__":
    main()
