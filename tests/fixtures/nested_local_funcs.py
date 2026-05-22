def outer(x: int) -> int:
    def inner(y: int) -> int:
        return y * 2

    def helper(z: int) -> int:
        return z + 1

    return inner(x) + helper(x)


def main() -> None:
    print(outer(5))
    print(outer(10))


if __name__ == "__main__":
    main()
