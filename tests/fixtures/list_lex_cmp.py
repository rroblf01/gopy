def main() -> None:
    a: list[int] = [1, 2, 3]
    b: list[int] = [1, 2, 4]
    print(a < b)
    print(a > b)
    print(a <= b)
    print(a >= b)
    print(a < [1, 2, 3, 0])
    print([1, 2] < [1, 2, 0])

    c: list[str] = ["a", "b"]
    d: list[str] = ["a", "c"]
    print(c < d)
    print(d > c)


if __name__ == "__main__":
    main()
