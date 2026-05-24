from functools import lru_cache, cache


@lru_cache
def fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)


@cache
def factorial(n: int) -> int:
    if n <= 1:
        return 1
    return n * factorial(n - 1)


def main() -> None:
    for i in range(10):
        print(fib(i))
    for i in range(1, 7):
        print(factorial(i))


if __name__ == "__main__":
    main()
