def main() -> None:
    grid = [(x, y) for x in [1, 2, 3] for y in [10, 20]]
    for p in grid:
        print(p[0])
        print(p[1])
    print("---")
    pairs = [x * y for x in [1, 2, 3] for y in [10, 20] if x + y > 12]
    for v in pairs:
        print(v)


if __name__ == "__main__":
    main()
