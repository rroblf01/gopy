def main() -> None:
    items: list[tuple[str, int, int]] = [("a", 1, 2), ("b", 3, 4), ("c", 5, 6)]
    for name, x, y in items:
        print(name, x + y)


if __name__ == "__main__":
    main()
