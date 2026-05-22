def fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)


def factorial(n: int) -> int:
    if n <= 1:
        return 1
    return n * factorial(n - 1)


def main() -> None:
    print(fib(10))
    print(factorial(10))
    for i in range(8):
        print(fib(i))


if __name__ == "__main__":
    main()
