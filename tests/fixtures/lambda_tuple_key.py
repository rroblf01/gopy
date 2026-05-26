def main() -> None:
    xs = [(1, "b"), (3, "a"), (2, "c")]
    by_key = sorted(xs, key=lambda p: p[0])
    print([t[0] for t in by_key])
    by_label = sorted(xs, key=lambda p: p[1])
    print([t[1] for t in by_label])
    # Confirm element subscript still typed for downstream string method
    first_label = by_label[0][1]
    print(first_label.upper())


if __name__ == "__main__":
    main()
