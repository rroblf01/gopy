def classify(v: int) -> str:
    match v:
        case 0:
            return "zero"
        case 1 | 2 | 3:
            return "small"
        case n if n < 10:
            return f"medium {n}"
        case _:
            return "large"


def main() -> None:
    print(classify(0))
    print(classify(2))
    print(classify(7))
    print(classify(100))


if __name__ == "__main__":
    main()
