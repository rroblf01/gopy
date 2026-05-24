def main() -> None:
    xs: list[int] = [1, 2, 3, 4, 5]

    first, *rest = xs
    print(first)
    print(rest)

    *head, last = xs
    print(head)
    print(last)

    a, *mid, z = xs
    print(a)
    print(mid)
    print(z)

    # Two-element list with star: middle is empty.
    p, *empty, q = [10, 20]
    print(p)
    print(empty)
    print(q)


if __name__ == "__main__":
    main()
