def first[T](xs: list[T]) -> T:
    return xs[0]


def last[T](xs: list[T]) -> T:
    return xs[len(xs) - 1]


def pair[A, B](a: A, b: B) -> list[A]:
    out: list[A] = [a]
    return out


def main() -> None:
    print(first([10, 20, 30]))
    print(first(["alpha", "beta"]))
    print(last([1, 2, 3, 4]))
    print(last(["x", "y", "z"]))
    p = pair(42, "hello")
    print(p[0])


if __name__ == "__main__":
    main()
