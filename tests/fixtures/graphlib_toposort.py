from graphlib import TopologicalSorter


def main() -> None:
    ts = TopologicalSorter()
    ts.add("c", "a", "b")
    ts.add("d", "c")
    order = ts.static_order()
    out: list[str] = []
    for item in order:
        out.append(str(item))
    print(",".join(out))


if __name__ == "__main__":
    main()
