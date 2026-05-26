def main() -> None:
    xs: list[int] = [0, 1, 2, 3, 4, 5]
    xs[::2] = [10, 20, 30]
    print(xs)

    ys: list[int] = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
    ys[1::2] = [100, 200, 300, 400, 500]
    print(ys)

    # Negative step
    zs: list[int] = [0, 1, 2, 3, 4]
    zs[::-1] = [10, 20, 30, 40, 50]
    print(zs)


if __name__ == "__main__":
    main()
