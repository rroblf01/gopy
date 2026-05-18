def main() -> None:
    # Generator expressions materialize as slices in the gopy shim, so
    # `for ... in` and `sum(...)` work identically to list comprehensions.
    nums: list[int] = [1, 2, 3, 4, 5]
    print(sum(x * x for x in nums))
    for v in (n * 10 for n in nums if n % 2 == 1):
        print(v)


if __name__ == "__main__":
    main()
