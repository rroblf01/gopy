def main() -> None:
    counts: dict[str, int] = {"a": 1, "b": 2}
    print(counts.get("a", 0))
    print(counts.get("missing", -1))
    print(counts.get("b", 999))


if __name__ == "__main__":
    main()
