import itertools


def main() -> None:
    xs: list[int] = [1, 2, 3, 4]
    pairs = list(itertools.pairwise(xs))
    for p in pairs:
        print(p[0], p[1])
    empty = list(itertools.pairwise([1]))
    print(len(empty))

    chunks = list(itertools.batched(xs, 2))
    for c in chunks:
        print(len(c), c[0])
    odd = list(itertools.batched([1, 2, 3, 4, 5], 3))
    print(len(odd))


if __name__ == "__main__":
    main()
