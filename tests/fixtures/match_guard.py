def classify(n: int) -> str:
    match n:
        case x if x < 0:
            return "negative"
        case 0:
            return "zero"
        case x if x < 10:
            return "small"
        case _:
            return "large"


def main() -> None:
    print(classify(-5))
    print(classify(0))
    print(classify(3))
    print(classify(100))


if __name__ == "__main__":
    main()
