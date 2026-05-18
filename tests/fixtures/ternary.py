def classify(n: int) -> str:
    return "positive" if n > 0 else ("zero" if n == 0 else "negative")


def main() -> None:
    print(classify(5))
    print(classify(0))
    print(classify(-3))
    x: int = 7
    y: int = 8 if x > 5 else 0
    print(y)


if __name__ == "__main__":
    main()
