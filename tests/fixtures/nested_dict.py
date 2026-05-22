def main() -> None:
    nested: dict[str, dict[str, int]] = {
        "a": {"x": 1, "y": 2},
        "b": {"x": 3, "y": 4},
    }
    print(nested["a"]["x"])
    print(nested["b"]["y"])
    for k in sorted(nested.keys()):
        for k2 in sorted(nested[k].keys()):
            print(k, k2, nested[k][k2])


if __name__ == "__main__":
    main()
