import random


def main() -> None:
    random.seed(42)
    xs: list[int] = [1, 2, 3, 4, 5]
    c = random.choice(xs)
    # c must be one of the elements; can't check exact (PRNG differs)
    print(c in xs)
    ys: list[int] = [1, 2, 3, 4, 5]
    random.shuffle(ys)
    print(len(ys))
    # ensure shuffle preserves elements
    ys.sort()
    for v in ys:
        print(v)
    s = random.sample([10, 20, 30, 40, 50], 3)
    print(len(s))
    for v in s:
        print(v in [10, 20, 30, 40, 50])
    u = random.uniform(0.0, 1.0)
    print(0.0 <= u <= 1.0)


if __name__ == "__main__":
    main()
