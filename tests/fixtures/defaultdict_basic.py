from collections import defaultdict


def main() -> None:
    # Annotation drives the actual map[K]V type; the int factory is
    # ignored because Go's zero value already covers the missing-key case.
    counts: dict[str, int] = defaultdict(int)
    for w in ["a", "b", "a", "c", "a", "b"]:
        counts[w] += 1
    print(counts["a"])
    print(counts["b"])
    print(counts["c"])
    # Missing key returns the zero value (0 for int).
    print(counts["z"])


if __name__ == "__main__":
    main()
