from functools import cache, lru_cache


@cache
def double(x: int) -> int:
    return x * 2


@lru_cache(maxsize=32)
def fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)


@cache
def add(a: int, b: int) -> int:
    return a + b


def main() -> None:
    print(double(5))
    print(double(7))
    print(fib(10))
    print(add(2, 3))
    print(add(10, 20))


if __name__ == "__main__":
    main()
