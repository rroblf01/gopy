def main() -> None:
    xs = [10, 20, 30, 40]
    for i in range(-len(xs), len(xs)):
        print(i, xs[i])
    s = "hello"
    for j in range(-len(s), len(s)):
        print(j, s[j])
    # nested
    grid: list[list[int]] = [[1, 2, 3], [4, 5, 6]]
    last = -1
    print(grid[last])
    print(grid[last][last])
    # expression-based index
    n = -1
    print(xs[n - 1])
    print(xs[n + 1])


if __name__ == "__main__":
    main()
