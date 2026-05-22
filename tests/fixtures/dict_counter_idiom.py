def main() -> None:
    counts: dict[str, int] = {}
    words = ["a", "b", "a", "c", "b", "a"]
    for w in words:
        counts[w] = counts.get(w, 0) + 1
    for k in sorted(counts.keys()):
        print(k, counts[k])


if __name__ == "__main__":
    main()
