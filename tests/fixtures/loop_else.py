def has(needle: int, hay: list[int]) -> str:
    for x in hay:
        if x == needle:
            return "found"
    else:
        return "not found"
    return "?"


def first_neg(start: int) -> int:
    i: int = start
    while i < 5:
        if i < 0:
            break
        i += 1
    else:
        return -1
    return i


def main() -> None:
    print(has(3, [1, 2, 3, 4]))
    print(has(9, [1, 2, 3, 4]))
    print(first_neg(0))


if __name__ == "__main__":
    main()
