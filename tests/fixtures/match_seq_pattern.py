def classify(xs: list[int]) -> str:
    match xs:
        case []:
            return "empty"
        case [0]:
            return "zero"
        case [1, 2]:
            return "pair-1-2"
        case [1, 2, 3]:
            return "triple-1-2-3"
        case _:
            return "other"


def main() -> None:
    print(classify([]))
    print(classify([0]))
    print(classify([1, 2]))
    print(classify([1, 2, 3]))
    print(classify([9, 9]))


if __name__ == "__main__":
    main()
