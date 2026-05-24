def main() -> None:
    xs: list[int] = [1, 2, 3]
    xs.extend([4, 5])
    print(xs)
    xs.insert(0, 0)
    print(xs)
    xs.remove(3)
    print(xs)
    ys: list[int] = [1, 2, 3, 2, 1, 4, 2]
    print(ys.index(2))
    print(ys.index(2, 2))
    print(ys.index(2, 2, 7))
    # str.replace with count
    s = "a-b-c-d"
    print(s.replace("-", "_", 2))
    print(s.replace("-", "/"))
    # str.zfill negative
    print("-42".zfill(6))
    print("+42".zfill(6))


if __name__ == "__main__":
    main()
