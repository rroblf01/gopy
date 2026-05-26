def main() -> None:
    d: dict[str, dict[str, int]] = {"a": {"x": 1, "y": 2}}
    inner = d.get("a", {})
    print(inner.get("x", -1))
    print(inner.get("z", -1))

    missing = d.get("nope", {})
    print(missing.get("x", -1))

    e: dict[str, list[int]] = {"a": [1, 2, 3]}
    xs = e.get("a", [])
    print(len(xs))
    ys = e.get("missing", [])
    print(len(ys))


if __name__ == "__main__":
    main()
