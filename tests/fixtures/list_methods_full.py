def main() -> None:
    xs: list[int] = [3, 1, 4, 1, 5, 9, 2, 6]
    # copy, sort, reverse
    ys: list[int] = list(xs)
    ys.sort()
    print(ys)
    ys.reverse()
    print(ys)
    print(xs)  # original untouched
    # extend
    xs.extend([100, 200])
    print(xs)
    # insert
    xs.insert(0, 999)
    print(xs[0])
    # pop
    last = xs.pop()
    print(last)
    print(xs[-1])
    # remove by value
    xs.remove(1)
    print(xs)
    # clear
    xs.clear()
    print(len(xs))


if __name__ == "__main__":
    main()
