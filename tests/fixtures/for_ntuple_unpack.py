def main() -> None:
    # Homogeneous list-of-lists: each element is `list[int]` so the
    # synthesized temp keeps its element type for indexing.
    rows: list[list[int]] = [[10, 20, 30], [40, 50, 60], [70, 80, 90]]
    for a, b, c in rows:
        print(a, b, c)

    # 4-name unpack works the same way.
    quads: list[list[int]] = [[1, 2, 3, 4], [5, 6, 7, 8]]
    for w, x, y, z in quads:
        print(w + x + y + z)


if __name__ == "__main__":
    main()
