def main() -> None:
    def fact(n: int) -> int:
        if n <= 1:
            return 1
        return n * fact(n - 1)

    print(fact(0))
    print(fact(1))
    print(fact(5))
    print(fact(7))

    def fib(n: int) -> int:
        if n < 2:
            return n
        return fib(n - 1) + fib(n - 2)

    print(fib(0))
    print(fib(1))
    print(fib(10))


if __name__ == "__main__":
    main()
