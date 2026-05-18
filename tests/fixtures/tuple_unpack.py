def main() -> None:
    # Parallel assign (literal both sides).
    a, b = 1, 2
    print(a)
    print(b)
    # Swap.
    a, b = b, a
    print(a)
    print(b)

    # enumerate(xs) → for i, x.
    names: list[str] = ["ada", "grace", "alan"]
    for i, n in enumerate(names):
        print(i)
        print(n)

    # zip(a, b) → for x, y.
    xs: list[int] = [1, 2, 3]
    ys: list[int] = [10, 20, 30]
    for x, y in zip(xs, ys):
        print(x + y)


if __name__ == "__main__":
    main()
