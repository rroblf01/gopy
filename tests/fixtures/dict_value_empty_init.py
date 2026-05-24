def main() -> None:
    groups: dict[str, list[int]] = {}
    groups["a"] = []
    groups["a"].append(1)
    groups["a"].append(2)
    groups["b"] = [3, 4]
    if "c" not in groups:
        groups["c"] = []
    groups["c"].append(99)
    for k in sorted(groups.keys()):
        print(k, groups[k])
    # nested dict[str, dict[str, int]]
    nested: dict[str, dict[str, int]] = {}
    nested["x"] = {}
    nested["x"]["a"] = 1
    nested["y"] = {"b": 2}
    for k in sorted(nested.keys()):
        for kk in sorted(nested[k].keys()):
            print(k, kk, nested[k][kk])


if __name__ == "__main__":
    main()
