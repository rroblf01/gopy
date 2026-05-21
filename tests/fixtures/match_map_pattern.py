def classify(d: dict[str, int]) -> str:
    match d:
        case {"x": 0}:
            return "zero-x"
        case {"x": 1, "y": 2}:
            return "x1y2"
        case {"x": 5}:
            return "x5"
        case _:
            return "other"


def main() -> None:
    print(classify({"x": 0, "y": 9}))
    print(classify({"x": 1, "y": 2}))
    print(classify({"x": 5}))
    print(classify({"x": 5, "extra": 99}))
    print(classify({"foo": 7}))


if __name__ == "__main__":
    main()
