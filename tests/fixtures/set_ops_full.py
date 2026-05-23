def main() -> None:
    a: set[int] = {1, 2, 3, 4}
    b: set[int] = {3, 4, 5, 6}
    inter = a & b
    print(sorted(inter))
    union = a | b
    print(sorted(union))
    diff = a - b
    print(sorted(diff))
    sym = a ^ b
    print(sorted(sym))


if __name__ == "__main__":
    main()
