def show(xs: list[int]) -> None:
    for v in xs:
        print(v)
    print("---")


def shows(xs: list[str]) -> None:
    for v in xs:
        print(v)
    print("---")


def main() -> None:
    xs: list[int] = [1, 2, 3]
    xs.extend([4, 5, 6])
    show(xs)
    xs.insert(0, 0)
    show(xs)
    xs.insert(3, 99)
    show(xs)
    xs.remove(99)
    show(xs)
    ys: list[str] = ["a", "b", "c"]
    ys.remove("b")
    shows(ys)
    ys.clear()
    print(len(ys))


if __name__ == "__main__":
    main()
