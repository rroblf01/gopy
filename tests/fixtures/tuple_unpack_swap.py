def main() -> None:
    a, b, c = 1, 2, 3
    print(a, b, c)
    a, b = b, a
    print(a, b)
    pair = (10, 20)
    x, y = pair
    print(x, y)
    nums = [100, 200, 300]
    p, q, r = nums
    print(p, q, r)


if __name__ == "__main__":
    main()
