def show(xs: list[int]) -> None:
    for v in xs:
        print(v)
    print("---")


def shows(xs: list[str]) -> None:
    for v in xs:
        print(v)
    print("---")


def main() -> None:
    zeros: list[int] = [0] * 5
    show(zeros)
    pair: list[int] = [1, 2] * 3
    show(pair)
    n: int = 4
    show([7] * n)
    show([9] * 0)
    shows(["x"] * 3)


if __name__ == "__main__":
    main()
