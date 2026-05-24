from itertools import combinations, product


def show(items: list[int]) -> str:
    parts: list[str] = []
    for x in items:
        parts.append(str(x))
    return "(" + ",".join(parts) + ")"


def main() -> None:
    nums: list[int] = [1, 2, 3, 4]

    # r-arity combinations.
    for combo in combinations(nums, 3):
        print(show(combo))

    # N-way product.
    a: list[int] = [1, 2]
    b: list[int] = [3, 4]
    c: list[int] = [5, 6]
    for triple in product(a, b, c):
        print(show(triple))

    # r=1 still works.
    for one in combinations(nums, 1):
        print(show(one))


if __name__ == "__main__":
    main()
