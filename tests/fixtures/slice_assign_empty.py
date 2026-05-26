def main() -> None:
    xs: list[int] = [1, 2, 3, 4, 5]
    xs[1:3] = [10, 20, 30]
    print(xs)

    ys: list[int] = [1, 2, 3, 4, 5]
    ys[1:4] = []
    print(ys)

    zs: list[str] = ["a", "b", "c"]
    zs[1:2] = []
    print(zs)


if __name__ == "__main__":
    main()
