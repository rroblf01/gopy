def main() -> None:
    nums = [1, 2, 3, 4, 5]
    # listcomp walrus in filter
    big = [s for n in nums if (s := str(n * 10))]
    print(big)
    # listcomp walrus in filter with bool cond
    out = [x for n in nums if (x := n * 2) > 4]
    print(out)
    # dictcomp walrus in filter and key/value
    d = {k: v for n in nums if (k := f"key{n}") and (v := n * n)}
    for kk in sorted(d.keys()):
        print(kk, d[kk])
    # nested walrus in elt
    seqs: list[list[int]] = [[1, 2], [3, 4, 5], [6]]
    lengths = [n for s in seqs if (n := len(s)) > 1]
    print(lengths)


if __name__ == "__main__":
    main()
