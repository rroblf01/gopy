from functools import lru_cache


@lru_cache
def fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)


@lru_cache(maxsize=128)
def double(n: int) -> int:
    return n * 2


def main() -> None:
    print(fib(10))
    print(double(21))


if __name__ == "__main__":
    main()
