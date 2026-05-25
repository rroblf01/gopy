calls = [0]


def mid() -> int:
    calls[0] += 1
    return 5


def main() -> None:
    # Chain `0 < mid() < 10` must evaluate mid() exactly once even
    # though it appears in two comparisons.
    if 0 < mid() < 10:
        print("ok")
    print(calls[0])


if __name__ == "__main__":
    main()
