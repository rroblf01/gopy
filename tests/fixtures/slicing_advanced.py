def main() -> None:
    xs: list[int] = [10, 20, 30, 40, 50, 60]
    for v in xs[-2:]:
        print(v)
    for v in xs[:-2]:
        print(v)
    for v in xs[::2]:
        print(v)
    for v in xs[1:5:2]:
        print(v)
    # Negative step: reverse iteration.
    for v in xs[::-1]:
        print(v)


if __name__ == "__main__":
    main()
