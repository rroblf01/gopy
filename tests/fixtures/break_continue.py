def first_even(xs: list[int]) -> int:
    for x in xs:
        if x % 2 != 0:
            continue
        return x
    return -1


def truncate_at(stop: int) -> int:
    total: int = 0
    for i in range(1, 100):
        if i == stop:
            break
        total += i
    return total


def main() -> None:
    print(first_even([1, 3, 5, 4, 7]))
    print(first_even([1, 3, 5]))
    print(truncate_at(10))


if __name__ == "__main__":
    main()
