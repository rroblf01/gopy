def main() -> None:
    nested: list[list[int]] = [[i * j for j in range(3)] for i in range(3)]
    print(nested)
    flat: list[int] = [x for row in nested for x in row]
    print(flat)
    # filter via outer cond
    filt: list[list[int]] = [[i * j for j in range(3)] for i in range(4) if i > 0]
    print(filt)
    # cross product
    pairs: list[int] = [a * b for a in range(3) for b in range(3) if a != b]
    print(pairs)


if __name__ == "__main__":
    main()
