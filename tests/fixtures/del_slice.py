def main() -> None:
    xs: list[int] = [1, 2, 3, 4, 5, 6]
    del xs[1:3]
    print(xs)

    ys: list[int] = [10, 20, 30, 40]
    del ys[:2]
    print(ys)

    zs: list[int] = [100, 200, 300]
    del zs[1:]
    print(zs)


if __name__ == "__main__":
    main()
