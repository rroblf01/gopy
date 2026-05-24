from collections import Counter, defaultdict


def main() -> None:
    items: list[str] = ["a", "b", "a", "c", "b", "a"]
    c = Counter(items)
    print(c["a"])
    print(c["b"])
    print(c["c"])
    print(c["missing"])

    groups: defaultdict[str, list[int]] = defaultdict(list)
    groups["x"].append(1)
    groups["y"].append(2)
    groups["x"].append(3)
    groups["y"].append(4)
    for k in sorted(groups.keys()):
        print(k, groups[k])

    counts: defaultdict[str, int] = defaultdict(int)
    for w in items:
        counts[w] += 1
    for k in sorted(counts.keys()):
        print(k, counts[k])


if __name__ == "__main__":
    main()
