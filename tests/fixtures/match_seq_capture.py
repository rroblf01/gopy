def classify(xs: list[int]) -> str:
    match xs:
        case []:
            return "empty"
        case [x]:
            return f"single={x}"
        case [first, *rest]:
            return f"first={first} rest_len={len(rest)}"
    return "?"


def head_last(xs: list[int]) -> str:
    match xs:
        case [head, *_, last]:
            return f"head={head} last={last}"
        case [only]:
            return f"only={only}"
        case []:
            return "empty"
    return "?"


def main() -> None:
    print(classify([]))
    print(classify([42]))
    print(classify([1, 2, 3, 4]))
    print(head_last([1, 2, 3]))
    print(head_last([10, 20]))
    print(head_last([7]))


if __name__ == "__main__":
    main()
