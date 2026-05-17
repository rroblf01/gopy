def main() -> None:
    counts: dict[str, int] = {"a": 1, "b": 2}
    counts["c"] = 3
    print(counts["a"])
    print(counts["b"])
    print(counts["c"])
    print(len(counts))


if __name__ == "__main__":
    main()
