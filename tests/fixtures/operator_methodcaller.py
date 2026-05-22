import operator


def main() -> None:
    print(operator.length_hint("hello"))
    print(operator.length_hint([1, 2, 3]))
    print(operator.index(42))
    print(operator.index(True))


if __name__ == "__main__":
    main()
