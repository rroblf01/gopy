def fib(n: int) -> int:
    a, b = 0, 1
    for _ in range(n):
        a, b = b, a + b
    return a


def fact(n: int) -> int:
    if n <= 1:
        return 1
    return n * fact(n - 1)


def main() -> None:
    for i in range(10):
        print(fib(i))
    for i in range(1, 8):
        print(fact(i))
    # _ in nested
    counter = 0
    for _ in range(3):
        for _ in range(2):
            counter += 1
    print(counter)


if __name__ == "__main__":
    main()
