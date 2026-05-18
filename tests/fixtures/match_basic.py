def classify(n: int) -> str:
    match n:
        case 0:
            return "zero"
        case 1 | 2 | 3:
            return "small"
        case _ if n < 0:
            return "negative"
        case _:
            return "big"


def main() -> None:
    print(classify(0))
    print(classify(2))
    print(classify(3))
    print(classify(100))
    print(classify(-5))


if __name__ == "__main__":
    main()
