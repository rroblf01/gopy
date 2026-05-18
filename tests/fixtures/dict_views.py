def main() -> None:
    d: dict[str, int] = {"a": 1, "b": 2, "c": 3}
    # Sum-of-values is order-independent so cross-runtime dict iteration
    # ordering doesn't break the fixture.
    ks: list[str] = d.keys()
    print(len(ks))
    vs: list[int] = d.values()
    print(sum(vs))
    # for-loop over .keys() / .values() works the same as direct dict iter.
    total: int = 0
    for v in d.values():
        total += v
    print(total)


if __name__ == "__main__":
    main()
