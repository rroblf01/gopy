def sum_to(n: int) -> int:
    total: int = 0
    for i in range(1, n + 1):
        total += i
    return total


def main() -> None:
    print(sum_to(10))
    print(sum_to(100))


if __name__ == "__main__":
    main()
