def main() -> None:
    s: set[int] = {1, 2, 3}
    s.add(4)
    s.add(2)  # dedup: no-op
    s.discard(99)  # missing: no-op
    s.discard(2)
    print(sorted(list(s)))

    t: set[str] = {"a"}
    t.add("b")
    t.add("a")
    print(sorted(list(t)))


if __name__ == "__main__":
    main()
