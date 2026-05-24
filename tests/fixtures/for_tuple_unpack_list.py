def main() -> None:
    # homogeneous int tuples
    coords = [(0, 0), (1, 2), (3, 4)]
    for x, y in coords:
        print(x, y)
    # typed list of tuples with explicit annotation
    result: list[tuple[str, int]] = []
    result.append(("x", 10))
    result.append(("y", 20))
    total: int = 0
    for k, v in result:
        print(k, v)
        total += v
    print("total:", total)


if __name__ == "__main__":
    main()
