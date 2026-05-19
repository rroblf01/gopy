from collections import OrderedDict


def main() -> None:
    d: dict[str, int] = OrderedDict()
    d["a"] = 1
    d["b"] = 2
    d["c"] = 3
    print(d["a"])
    print(d["b"])
    print(d["c"])
    print(len(d))


if __name__ == "__main__":
    main()
