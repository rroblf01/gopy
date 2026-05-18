def main() -> None:
    xs: list[int] = [1, 2, 3]
    ys: list[int] = [4, 5]
    xs += ys
    for v in xs:
        print(v)
    print(len(xs))


if __name__ == "__main__":
    main()
