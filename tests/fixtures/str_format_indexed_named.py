def main() -> None:
    print("{0} and {1}".format("a", "b"))
    print("{1} before {0}".format("a", "b"))
    print("{0}-{0}".format("x"))
    print("{name}={val}".format(name="x", val=42))
    print("{name}: {n:5d}".format(name="abc", n=3))
    print("{}, {}, {}".format("p", "q", "r"))
    print("{:>8}".format("hi"))
    print("{0:*^7}".format("Y"))
    # print unpack
    args: list[int] = [1, 2, 3]
    print(*args)
    # any/all edge
    print(any([False, False, True]))
    print(all([True, True, False]))
    print(any([False, False]))
    print(all([]))


if __name__ == "__main__":
    main()
