def main() -> None:
    # Tuple literals compile down to slices, so indexing and iteration
    # work the same as for lists. Immutability isn't enforced.
    pair: list[int] = (3, 5)
    print(pair[0])
    print(pair[1])
    triple: list[str] = ("a", "b", "c")
    for s in triple:
        print(s)


if __name__ == "__main__":
    main()
