class Box:
    def __init__(self, v: int) -> None:
        self.v = v


def main() -> None:
    nums: list[int] = [3, 1, 4, 1, 5, 9, 2, 6]
    r: list[int] = reversed(nums)
    for v in r:
        print(v)

    print(abs(-7))
    print(abs(5))
    print(abs(-3.5))
    print(round(2.7))
    print(round(2.3))
    print(round(5))

    b: Box = Box(42)
    print(isinstance(b, Box))
    print(isinstance(42, int))
    print(isinstance("hi", str))
    print(isinstance(b, int))


if __name__ == "__main__":
    main()
