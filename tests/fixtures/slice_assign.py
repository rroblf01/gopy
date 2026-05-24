def main() -> None:
    xs: list[int] = [1, 2, 3, 4, 5]
    xs[1:3] = [10, 20, 30]
    print(xs)

    ys: list[int] = [1, 2, 3, 4]
    ys[:2] = [99]
    print(ys)

    zs: list[int] = [1, 2, 3, 4]
    zs[2:] = [88, 77]
    print(zs)


if __name__ == "__main__":
    main()
