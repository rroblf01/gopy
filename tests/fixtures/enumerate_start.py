def main() -> None:
    xs: list[str] = ["a", "b", "c"]
    for i, v in enumerate(xs):
        print(i)
        print(v)
    print("---")
    for i, v in enumerate(xs, 10):
        print(i)
        print(v)
    print("---")
    for i, v in enumerate(xs, start=5):
        print(i)
        print(v)


if __name__ == "__main__":
    main()
