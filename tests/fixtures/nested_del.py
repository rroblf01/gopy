def main() -> None:
    d: dict[str, list[int]] = {"a": [10, 20, 30, 40]}
    del d["a"][1]
    print(d["a"][0])
    print(d["a"][1])
    print(len(d["a"]))

    grid: list[list[int]] = [[1, 2, 3], [4, 5, 6]]
    del grid[0][1]
    print(grid[0][0])
    print(grid[0][1])
    print(len(grid[0]))


if __name__ == "__main__":
    main()
