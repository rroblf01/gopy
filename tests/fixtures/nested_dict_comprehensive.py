def main() -> None:
    nested: dict[str, dict[str, int]] = {
        "x": {"a": 1, "b": 2},
        "y": {"a": 3, "b": 4},
    }
    for outer in sorted(nested.keys()):
        inner = nested[outer]
        for ik in sorted(inner.keys()):
            print(outer, ik, inner[ik])
    # update inner
    nested["x"]["c"] = 99
    for ik in sorted(nested["x"].keys()):
        print("x", ik, nested["x"][ik])
    # multi-level dict comp
    sums: dict[str, int] = {}
    for k, v in nested.items():
        sums[k] = sum(v.values())
    for k in sorted(sums.keys()):
        print(k, sums[k])


if __name__ == "__main__":
    main()
