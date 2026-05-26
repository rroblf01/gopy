def main() -> None:
    def use(n: int) -> int:
        return helper(n) + 1

    def helper(n: int) -> int:
        return n * 2

    print(use(5))
    print(use(10))


if __name__ == "__main__":
    main()
