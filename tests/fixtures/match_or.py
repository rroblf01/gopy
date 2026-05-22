def label(n: int) -> str:
    match n:
        case 1 | 2 | 3:
            return "low"
        case 4 | 5 | 6:
            return "mid"
        case 7 | 8 | 9:
            return "high"
        case _:
            return "other"


def main() -> None:
    print(label(2))
    print(label(5))
    print(label(8))
    print(label(0))
    print(label(10))


if __name__ == "__main__":
    main()
