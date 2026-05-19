def show(xs: list[int]) -> None:
    for v in xs:
        print(v)
    print("---")


def main() -> None:
    xs: list[int] = [3, 1, 4, 1, 5, 9, 2, 6]
    xs.sort()
    show(xs)
    xs.sort(reverse=True)
    show(xs)
    xs.reverse()
    show(xs)
    ys: list[str] = ["b", "a", "c"]
    ys.sort()
    for v in ys:
        print(v)


if __name__ == "__main__":
    main()
