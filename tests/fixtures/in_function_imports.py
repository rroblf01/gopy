def main() -> None:
    from itertools import chain, repeat, accumulate
    cc = list(chain([1, 2, 3], [4, 5]))
    print(cc)
    rep = list(repeat("x", 4))
    print(rep)
    acc = list(accumulate([1, 2, 3, 4]))
    print(acc)
    # math import inside fn
    import math
    print(round(math.pi, 4))
    # collections inside fn
    from collections import Counter
    c = Counter(["a", "b", "a", "c"])
    print(c["a"])
    print(c["z"])


if __name__ == "__main__":
    main()
